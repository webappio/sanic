from datetime import datetime

from flask import Flask, render_template
from redis import Redis

app = Flask(__name__)
redis = Redis(host='redis', retry_on_timeout=True)


@app.route('/')
def index():
    timestamp_id = redis.get('TIMESTAMP_ID') or 0
    return render_template('index.html', timestamp=timestamp_id)


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=80)
