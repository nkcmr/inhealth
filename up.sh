#!/bin/bash
mkdir -p data/prometheus/
mkdir -p data/grafana/
docker-compose up -d --build
