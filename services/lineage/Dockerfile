FROM python:3.11-slim

# librdkafka is required by confluent-kafka at runtime.
# Installing it from apt is faster and more reliable than building from source.
RUN apt-get update && apt-get install -y --no-install-recommends \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -r -s /bin/false mzigo

WORKDIR /app

COPY pyproject.toml .
RUN pip install --no-cache-dir -e .

COPY . .

USER mzigo

EXPOSE 9102

HEALTHCHECK --interval=15s --timeout=5s --start-period=30s \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:9102/metrics')" || exit 1

ENTRYPOINT ["python3", "main.py"]
