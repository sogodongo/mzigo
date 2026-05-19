FROM node:20-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

COPY . .

# Service URLs are injected at runtime via Kubernetes ConfigMap.
# We set empty placeholders here so the build does not fail on missing vars.
ENV CONTRACTS_SERVICE_URL=http://contracts:8081
ENV ANALYZER_SERVICE_URL=http://analyzer:8083
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

FROM node:20-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system mzigo && adduser --system --ingroup mzigo mzigo

COPY --from=builder /app/public ./public
COPY --from=builder --chown=mzigo:mzigo /app/.next/standalone ./
COPY --from=builder --chown=mzigo:mzigo /app/.next/static ./.next/static

USER mzigo

EXPOSE 3000

HEALTHCHECK --interval=15s --timeout=5s --start-period=30s \
    CMD wget -qO- http://localhost:3000/api/health || exit 1

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
