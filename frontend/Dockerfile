# Build stage
FROM node:14 as build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Production stage (no Nginx)
FROM node:14
WORKDIR /app
RUN npm install -g serve
COPY --from=build /app/build ./build
EXPOSE 82
CMD ["serve", "-s", "build", "-l", "82"]
