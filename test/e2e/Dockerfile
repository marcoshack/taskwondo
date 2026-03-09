FROM mcr.microsoft.com/playwright:v1.58.2-noble
WORKDIR /e2e
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENTRYPOINT ["npx", "playwright", "test"]
