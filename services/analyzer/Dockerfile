FROM python:3.11-slim

RUN useradd -r -s /bin/false mzigo

WORKDIR /app

COPY pyproject.toml .
RUN pip install --no-cache-dir -e .

COPY . .

USER mzigo

EXPOSE 8083 9103

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8083/healthz')" || exit 1

ENTRYPOINT ["python3", "main.py"]
