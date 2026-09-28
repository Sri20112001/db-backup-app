package agent

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// nativeMagic identifies our driver-based MongoDB archive. Unlike mongodump
// output (gzip), this format needs no external tools to create or restore.
const nativeMagic = "VGM1"

// mongoConnect dials with a short timeout for handshake.
func mongoConnect(ctx context.Context, uri string) (*mongo.Client, error) {
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client, err := mongo.Connect(c, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, redactMongoURI(uri, err.Error())
	}
	ping, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(ping, readpref.Primary()); err != nil {
		return nil, redactMongoURI(uri, err.Error())
	}
	return client, nil
}

// redactMongoURI replaces credentials in text: mongodb://user:pass@ -> mongodb://***@
func redactMongoURI(uri string, msg string) error {
	redacted := msg
	if i := strings.LastIndex(redacted, "@"); i >= 0 {
		if j := strings.Index(redacted, "://"); j >= 0 && j < i {
			redacted = redacted[:j+3] + "***" + redacted[i:]
		}
	}
	_ = uri
	return fmt.Errorf("%s", firstLine(redacted))
}

func writeUvarint(w io.Writer, v uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], v)
	_, err := w.Write(buf[:n])
	return err
}

func writeFrame(w io.Writer, data []byte) error {
	if err := writeUvarint(w, uint64(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// BackupMongoNative dumps every regular collection of dbName (documents as
// raw BSON plus index specs) into a single streaming archive. Memory stays
// flat: one document in flight at a time.
func BackupMongoNative(ctx context.Context, uri, dbName string) (string, int64, error) {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return "", 0, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	tmp, err := os.CreateTemp("", "vg-mongo-*.vgm")
	if err != nil {
		return "", 0, fmt.Errorf("create temp archive: %w", err)
	}
	archivePath := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(archivePath)
		}
	}()
	if _, err := tmp.Write([]byte(nativeMagic)); err != nil {
		return "", 0, err
	}

	// Regular collections only: no system.*, no views (a view would restore
	// as a dead collection).
	filter := bson.M{
		"name": bson.M{"$not": bson.M{"$regex": "^system\\."}},
		"type": "collection",
	}
	names, err := db.ListCollectionNames(ctx, filter)
	if err != nil {
		return "", 0, fmt.Errorf("list collections: %w", err)
	}
	if err := writeUvarint(tmp, uint64(len(names))); err != nil {
		return "", 0, err
	}

	for _, name := range names {
		if err := writeFrame(tmp, []byte(name)); err != nil {
			return "", 0, err
		}
		// Index specs as raw documents, minus server-stamped fields the
		// createIndexes command rejects.
		cur, err := db.Collection(name).Indexes().List(ctx)
		if err != nil {
			return "", 0, fmt.Errorf("list indexes %s: %w", name, err)
		}
		var specs [][]byte
		for cur.Next(ctx) {
			var doc bson.D
			if err := cur.Decode(&doc); err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
			clean := make(bson.D, 0, len(doc))
			for _, e := range doc {
				if e.Key == "ns" || e.Key == "v" {
					continue
				}
				clean = append(clean, e)
			}
			raw, err := bson.Marshal(clean)
			if err != nil {
				cur.Close(ctx)
				return "", 0, err
			}
			specs = append(specs, raw)
		}
		cur.Close(ctx)
		if err := cur.Err(); err != nil {
			return "", 0, err
		}
		if err := writeUvarint(tmp, uint64(len(specs))); err != nil {
			return "", 0, err
		}
		for _, s := range specs {
			if err := writeFrame(tmp, s); err != nil {
				return "", 0, err
			}
		}

		// Documents, streamed.
		dcur, err := db.Collection(name).Find(ctx, bson.D{})
		if err != nil {
			return "", 0, fmt.Errorf("find %s: %w", name, err)
		}
		// Count first would double-scan; instead buffer lengths via a tee:
		// write docs to a side buffer, then frame it. Collections are read
		// once; large ones spill only as raw BSON in memory per doc since
		// we encode doc-by-doc into a growing buffer... to keep it truly
		// streaming we store doc frames into a temp file.
		docTmp, err := os.CreateTemp("", "vg-mongodocs-*.bin")
		if err != nil {
			dcur.Close(ctx)
			return "", 0, err
		}
		docPath := docTmp.Name()
		var nDocs uint64
		scanErr := func() error {
			defer dcur.Close(ctx)
			defer docTmp.Close()
			for dcur.Next(ctx) {
				var raw bson.Raw
				if err := dcur.Decode(&raw); err != nil {
					return err
				}
				if err := writeFrame(docTmp, raw); err != nil {
					return err
				}
				nDocs++
			}
			return dcur.Err()
		}()
		if scanErr != nil {
			os.Remove(docPath)
			return "", 0, fmt.Errorf("scan %s: %w", name, scanErr)
		}
		if err := writeUvarint(tmp, nDocs); err != nil {
			os.Remove(docPath)
			return "", 0, err
		}
		// Splice the doc frames into the archive.
		df, err := os.Open(docPath)
		if err != nil {
			os.Remove(docPath)
			return "", 0, err
		}
		_, err = io.Copy(tmp, df)
		df.Close()
		os.Remove(docPath)
		if err != nil {
			return "", 0, err
		}
	}

	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	st, err := os.Stat(archivePath)
	if err != nil {
		os.Remove(archivePath)
		return "", 0, err
	}
	ok = true
	return archivePath, st.Size(), nil
}

type frameReader struct {
	br *byteReader
}

func (f *frameReader) next() ([]byte, error) {
	n, err := binary.ReadUvarint(f.br)
	if err != nil {
		return nil, err
	}
	if n > 256<<20 {
		return nil, fmt.Errorf("corrupt frame length %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(f.br, buf); err != nil {
		return nil, fmt.Errorf("truncated frame: %w", err)
	}
	return buf, nil
}

func (f *frameReader) nextUint() (uint64, error) {
	return binary.ReadUvarint(f.br)
}

// RestoreMongoNative replays a native archive into targetDB (created
// implicitly on first insert): collections, documents in batches, indexes.
func RestoreMongoNative(ctx context.Context, uri, targetDB, archivePath string) error {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(targetDB)

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	br := &byteReader{r: f}
	fr := &frameReader{br: br}

	magic := make([]byte, len(nativeMagic))
	if _, err := io.ReadFull(f, magic); err != nil || string(magic) != nativeMagic {
		return fmt.Errorf("not a VaultGuard mongo archive")
	}
	nColls, err := fr.nextUint()
	if err != nil {
		return fmt.Errorf("corrupt archive header: %w", err)
	}
	for i := uint64(0); i < nColls; i++ {
		nameRaw, err := fr.next()
		if err != nil {
			return fmt.Errorf("collection %d: %w", i, err)
		}
		name := string(nameRaw)
		nIdx, err := fr.nextUint()
		if err != nil {
			return fmt.Errorf("indexes %s: %w", name, err)
		}
		var idxModels []mongo.IndexModel
		for k := uint64(0); k < nIdx; k++ {
			spec, err := fr.next()
			if err != nil {
				return fmt.Errorf("index spec %s: %w", name, err)
			}
			var keys bson.D
			if err := bson.Unmarshal(spec, &keys); err != nil {
				return fmt.Errorf("bad index spec: %w", err)
			}
			// Split key doc from options (name=, unique=, ...).
			var keyDoc bson.D
			var optDoc bson.D
			for _, e := range keys {
				if e.Key == "key" {
					if kd, ok := e.Value.(bson.D); ok {
						keyDoc = kd
						continue
					}
				}
				optDoc = append(optDoc, e)
			}
			if len(keyDoc) == 0 {
				keyDoc = keys // spec already is the key (e.g. {_id:1})
				optDoc = nil
			}
			m := mongo.IndexModel{Keys: keyDoc}
			if len(optDoc) > 0 {
				m.Options = options.Index()
				for _, e := range optDoc {
					switch e.Key {
					case "name":
						if s, ok := e.Value.(string); ok {
							m.Options.SetName(s)
						}
					case "unique":
						if b, ok := e.Value.(bool); ok {
							m.Options.SetUnique(b)
						}
					case "sparse":
						if b, ok := e.Value.(bool); ok {
							m.Options.SetSparse(b)
						}
					case "expireAfterSeconds":
						switch v := e.Value.(type) {
						case int32:
							m.Options.SetExpireAfterSeconds(v)
						case int64:
							m.Options.SetExpireAfterSeconds(int32(v))
						}
					}
				}
			}
			idxModels = append(idxModels, m)
		}
		nDocs, err := fr.nextUint()
		if err != nil {
			return fmt.Errorf("doc count %s: %w", name, err)
		}
		coll := db.Collection(name)
		const batchSize = 500
		batch := make([]interface{}, 0, batchSize)
		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			if _, err := coll.InsertMany(ctx, batch); err != nil {
				return fmt.Errorf("insert %s: %w", name, err)
			}
			batch = batch[:0]
			return nil
		}
		for d := uint64(0); d < nDocs; d++ {
			raw, err := fr.next()
			if err != nil {
				return fmt.Errorf("doc %d/%s: %w", d, name, err)
			}
			var doc bson.D
			if err := bson.Unmarshal(raw, &doc); err != nil {
				return fmt.Errorf("decode doc: %w", err)
			}
			batch = append(batch, doc)
			if len(batch) >= batchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		}
		if err := flush(); err != nil {
			return err
		}
		if len(idxModels) > 0 {
			if _, err := coll.Indexes().CreateMany(ctx, idxModels); err != nil {
				return fmt.Errorf("create indexes %s: %w", name, err)
			}
		}
	}
	return nil
}

// mongoDocCount returns the total document count across regular collections.
func mongoDocCount(ctx context.Context, uri, dbName string) (int64, error) {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return 0, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)
	names, err := db.ListCollectionNames(ctx, bson.M{
		"name": bson.M{"$not": bson.M{"$regex": "^system\\."}},
		"type": "collection",
	})
	if err != nil {
		return 0, redactMongoURI(uri, err.Error())
	}
	var total int64
	for _, name := range names {
		n, err := db.Collection(name).EstimatedDocumentCount(ctx)
		if err != nil {
			return 0, redactMongoURI(uri, err.Error())
		}
		total += n
	}
	return total, nil
}

// PrecheckMongo fails when the database holds no documents — dumping an
// empty database would record a successful backup of nothing.
func PrecheckMongo(ctx context.Context, uri, dbName string) error {
	n, err := mongoDocCount(ctx, uri, dbName)
	if err != nil {
		return fmt.Errorf("pre-check %s: %w", dbName, err)
	}
	if n == 0 {
		return fmt.Errorf("pre-check %s: database contains no documents to back up", dbName)
	}
	return nil
}

// DiscoverMongoNative lists user databases via the driver (no mongosh needed).
func DiscoverMongoNative(ctx context.Context, uri string) ([]string, error) {
	client, err := mongoConnect(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect(context.Background())
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	names, err := client.ListDatabaseNames(c, bson.D{})
	if err != nil {
		return nil, redactMongoURI(uri, err.Error())
	}
	var out []string
	for _, n := range names {
		if n == "admin" || n == "local" || n == "config" {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("connected but no databases returned")
	}
	return out, nil
}

// byteReader adapts *os.File to io.ByteReader.
type byteReader struct {
	r io.Reader
}

func (b *byteReader) Read(p []byte) (int, error) { return b.r.Read(p) }

func (b *byteReader) ReadByte() (byte, error) {
	var one [1]byte
	if _, err := io.ReadFull(b.r, one[:]); err != nil {
		return 0, err
	}
	return one[0], nil
}
