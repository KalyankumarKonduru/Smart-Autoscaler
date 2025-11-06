# --- Build stage ---
    FROM node:20-alpine AS build
    WORKDIR /app
    COPY package*.json ./
    ENV NODE_ENV=development
    RUN npm ci --include=dev
    COPY . .
    RUN npm run build
    
    # --- Runtime stage ---
    FROM nginx:1.27-alpine
    ENV PORT=8080
    # Use a template so nginx listens on the Cloud Run PORT
    RUN rm /etc/nginx/conf.d/default.conf
    COPY nginx.conf /etc/nginx/templates/default.conf.template
    COPY --from=build /app/dist /usr/share/nginx/html
    # Render $PORT into nginx conf, then run nginx in foreground
    CMD ["sh","-c","envsubst '$PORT' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf && exec nginx -g 'daemon off;'"]
    