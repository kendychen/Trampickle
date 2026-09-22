FROM node:20-alpine AS build
WORKDIR /app
COPY package.json ./
COPY scripts/build.mjs ./scripts/build.mjs
COPY giao-trinh-sua-vot ./giao-trinh-sua-vot
COPY sitemap.xml robots.txt ./
RUN node scripts/build.mjs

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx","-g","daemon off;"]