FROM python:3.9-slim

RUN apt-get update && \
    apt-get install -y git && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN git clone https://github.com/initstring/cloud_enum.git . && \
    pip install -r requirements.txt

ENTRYPOINT ["python", "cloud_enum.py"] 