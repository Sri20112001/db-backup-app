package agent

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportMongo backs up dbName in JSON or CSV (native, no tools required),
// gzipping the result when compressed (COMPRESSED job mode).
func ExportMongo(ctx context.Context, uri, dbName, format string, compressed bool) (string, int64, error) {
	var (
		path string
		size int64
		err  error
		ext  string
	)
	switch format {
	case "JSON":
		path, size, err = BackupMongoJSON(ctx, uri, dbName)
		ext = ".json"
	case "CSV":
		path, size, err = BackupMongoCSV(ctx, uri, dbName)
		ext = ".csv"
	default:
		return "", 0, fmt.Errorf("unknown export format %q", format)
	}
	if err != nil {
		return "", 0, err
	}
	_ = ext
	if !compressed {
		return path, size, nil
	}
	gzPath := path + ".gz"
	if err := GzipFile(path, gzPath); err != nil {
		os.Remove(path)
		return "", 0, fmt.Errorf("compress export: %w", err)
	}
	os.Remove(path)
	st, err := os.Stat(gzPath)
	if err != nil {
		os.Remove(gzPath)
		return "", 0, err
	}
	return gzPath, st.Size(), nil
}

// mongoCollectionNames returns regular (non-system, non-view) collections.
func mongoCollectionNames(ctx context.Context, uri, dbName string) ([]string, error) {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	names, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{
		"name": bson.M{"$not": bson.M{"$regex": "^system\\."}},
		"type": "collection",
	})
	if err != nil {
		return nil, redactMongoURI(uri, err.Error())
	}
	return names, nil
}

// BackupMongoJSON writes one relaxed-EJSON document per line, grouped under
// {"$vg_collection": name} marker lines so restore maps lines back.
func BackupMongoJSON(ctx context.Context, uri, dbName string) (string, int64, error) {
	if err := PrecheckMongo(ctx, uri, dbName); err != nil {
		return "", 0, err
	}
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return "", 0, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	names, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{
		"name": bson.M{"$not": bson.M{"$regex": "^system\\."}},
		"type": "collection",
	})
	if err != nil {
		return "", 0, redactMongoURI(uri, err.Error())
	}

	tmp, err := os.CreateTemp("", "vg-mongojson-*.json")
	if err != nil {
		return "", 0, err
	}
	path := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	w := bufio.NewWriter(tmp)
	for _, name := range names {
		if _, err := fmt.Fprintf(w, "{\"$vg_collection\":%q}\n", name); err != nil {
			return "", 0, err
		}
		cur, err := db.Collection(name).Find(ctx, bson.D{})
		if err != nil {
			return "", 0, redactMongoURI(uri, err.Error())
		}
		for cur.Next(ctx) {
			var doc bson.D
			if err := cur.Decode(&doc); err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
			ext, err := bson.MarshalExtJSON(doc, false, false)
			if err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
			if _, err := w.Write(append(ext, '\n')); err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
		}
		cerr := cur.Err()
		cur.Close(ctx)
		if cerr != nil {
			return "", 0, redactMongoURI(uri, cerr.Error())
		}
	}
	if err := w.Flush(); err != nil {
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	st, err := os.Stat(path)
	if err != nil {
		os.Remove(path)
		return "", 0, err
	}
	ok = true
	return path, st.Size(), nil
}

// RestoreMongoJSON replays a marker-grouped EJSON-lines file into targetDB.
func RestoreMongoJSON(ctx context.Context, uri, targetDB, path string) error {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(targetDB)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	const batchSize = 500
	var coll string
	var batch []interface{}
	flush := func() error {
		if len(batch) == 0 || coll == "" {
			return nil
		}
		if _, err := db.Collection(coll).InsertMany(ctx, batch); err != nil {
			return fmt.Errorf("insert %s: %w", coll, redactMongoURI(uri, firstLine(err.Error())))
		}
		batch = batch[:0]
		return nil
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var probe struct {
			Coll string `json:"$vg_collection"`
		}
		// Marker detection uses encoding/json: the line is plain JSON and
		// bson.UnmarshalExtJSON would ignore the json tag.
		if err := json.Unmarshal([]byte(line), &probe); err != nil {
			return fmt.Errorf("parse line: %w", err)
		}
		if probe.Coll != "" {
			if err := flush(); err != nil {
				return err
			}
			coll = probe.Coll
			continue
		}
		if coll == "" {
			return fmt.Errorf("document before any collection marker")
		}
		var doc bson.D
		if err := bson.UnmarshalExtJSON([]byte(line), false, &doc); err != nil {
			return fmt.Errorf("parse document: %w", err)
		}
		batch = append(batch, doc)
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read dump: %w", err)
	}
	return flush()
}

// flattenBSON flattens a document to dotted paths with scalar strings.
// Nested documents recurse; arrays and other composites become JSON strings.
func flattenBSON(prefix string, doc bson.D, out map[string]string) {
	for _, e := range doc {
		key := e.Key
		if prefix != "" {
			key = prefix + "." + e.Key
		}
		switch v := e.Value.(type) {
		case bson.D:
			flattenBSON(key, v, out)
		case bson.M:
			flattenBSON(key, bsonDFromM(v), out)
		case primitive.ObjectID:
			out[key] = v.Hex()
		case primitive.DateTime:
			out[key] = v.Time().UTC().Format("2006-01-02T15:04:05.999Z07:00")
		case string:
			out[key] = v
		case bool:
			out[key] = strconv.FormatBool(v)
		case int32:
			out[key] = strconv.FormatInt(int64(v), 10)
		case int64:
			out[key] = strconv.FormatInt(v, 10)
		case float64:
			out[key] = strconv.FormatFloat(v, 'f', -1, 64)
		case nil:
			out[key] = ""
		default:
			if raw, err := bson.MarshalExtJSON(e.Value, false, false); err == nil {
				out[key] = string(raw)
			} else {
				out[key] = ""
			}
		}
	}
}

func bsonDFromM(m bson.M) bson.D {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(bson.D, 0, len(m))
	for _, k := range keys {
		out = append(out, bson.E{Key: k, Value: m[k]})
	}
	return out
}

// BackupMongoCSV writes a flattened CSV (header + rows) across all regular
// collections, prefixed by per-collection marker comment lines so restore
// maps rows back. Values are strings by nature of CSV.
func BackupMongoCSV(ctx context.Context, uri, dbName string) (string, int64, error) {
	if err := PrecheckMongo(ctx, uri, dbName); err != nil {
		return "", 0, err
	}
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return "", 0, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	names, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{
		"name": bson.M{"$not": bson.M{"$regex": "^system\\."}},
		"type": "collection",
	})
	if err != nil {
		return "", 0, redactMongoURI(uri, err.Error())
	}

	tmp, err := os.CreateTemp("", "vg-mongocsv-*.csv")
	if err != nil {
		return "", 0, err
	}
	path := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	w := csv.NewWriter(tmp)
	for _, name := range names {
		// Pass 1: header union.
		header := []string{}
		seen := map[string]bool{}
		cur, err := db.Collection(name).Find(ctx, bson.D{})
		if err != nil {
			return "", 0, redactMongoURI(uri, err.Error())
		}
		var docs []map[string]string
		for cur.Next(ctx) {
			var doc bson.D
			if err := cur.Decode(&doc); err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
			flat := map[string]string{}
			flattenBSON("", doc, flat)
			for k := range flat {
				if !seen[k] {
					seen[k] = true
					header = append(header, k)
				}
			}
			docs = append(docs, flat)
		}
		cerr := cur.Err()
		cur.Close(ctx)
		if cerr != nil {
			return "", 0, redactMongoURI(uri, cerr.Error())
		}
		sort.Strings(header)
		// Marker row + header + data rows.
		if err := w.Write([]string{"$vg_collection", name}); err != nil {
			return "", 0, err
		}
		if err := w.Write(header); err != nil {
			return "", 0, err
		}
		for _, flat := range docs {
			row := make([]string, len(header))
			for i, h := range header {
				row[i] = flat[h]
			}
			if err := w.Write(row); err != nil {
				return "", 0, err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return "", 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	st, err := os.Stat(path)
	if err != nil {
		os.Remove(path)
		return "", 0, err
	}
	ok = true
	return path, st.Size(), nil
}

// RestoreMongoCSV replays a flattened CSV into targetDB. Every value lands
// as a string (except _id, revived to ObjectID when it parses) — use ARCHIVE
// or JSON for exact restores.
func RestoreMongoCSV(ctx context.Context, uri, targetDB, path string) error {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(targetDB)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	const batchSize = 500
	var coll string
	var header []string
	var batch []interface{}
	flush := func() error {
		if len(batch) == 0 || coll == "" {
			return nil
		}
		if _, err := db.Collection(coll).InsertMany(ctx, batch); err != nil {
			return fmt.Errorf("insert %s: %w", coll, redactMongoURI(uri, firstLine(err.Error())))
		}
		batch = batch[:0]
		return nil
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parse csv: %w", err)
		}
		if len(row) == 2 && row[0] == "$vg_collection" {
			if err := flush(); err != nil {
				return err
			}
			coll = row[1]
			header = nil
			continue
		}
		if coll == "" {
			continue
		}
		if header == nil {
			header = append([]string{}, row...)
			continue
		}
		doc := make(bson.D, 0, len(header))
		for i, h := range header {
			var v interface{} = ""
			if i < len(row) {
				v = row[i]
			}
			// Revive ObjectIDs so _id keeps its type when possible.
			if h == "_id" {
				if s, ok := v.(string); ok {
					if oid, err := primitive.ObjectIDFromHex(s); err == nil {
						v = oid
					}
				}
			}
			doc = append(doc, bson.E{Key: h, Value: v})
		}
		batch = append(batch, doc)
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}
