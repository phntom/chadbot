FROM python
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY chadbot chadbot
EXPOSE 7357
ENV PYTHONPATH=/app
CMD ["python", "/app/chadbot/__main__.py"]
