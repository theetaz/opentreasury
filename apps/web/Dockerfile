# Build from the repository root so the pnpm workspace resolves:
#   docker build -f apps/web/Dockerfile .
FROM node:20-slim AS build
WORKDIR /repo
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY packages/types/package.json packages/types/package.json
COPY packages/ui/package.json packages/ui/package.json
RUN corepack pnpm install --filter @opentreasury/web --filter @opentreasury/types --filter @opentreasury/ui --frozen-lockfile
COPY apps/web apps/web
COPY packages packages
ARG VITE_CORE_API_URL
ENV VITE_CORE_API_URL=${VITE_CORE_API_URL}
RUN corepack pnpm --filter @opentreasury/web build

# Serve
FROM nginx:1.27-alpine
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /repo/apps/web/dist /usr/share/nginx/html
EXPOSE 80
