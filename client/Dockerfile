# Build stage
FROM node:20-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Serve stage
FROM nginx:alpine
# Vite builds with base '/vaultguard/', so serve dist under the /vaultguard/
# subpath to match nginx.conf (alias /usr/share/nginx/html/vaultguard/).
COPY --from=builder /app/dist /usr/share/nginx/html/vaultguard
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
