# Python relay for Amvera (build context: repo root).
FROM python:3.11-slim
WORKDIR /app
COPY deploy/amvera/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY deploy/amvera/app ./app
COPY deploy/amvera/game ./game
COPY deploy/amvera/run.py .
ENV PORT=80
EXPOSE 80
CMD ["python", "run.py"]
