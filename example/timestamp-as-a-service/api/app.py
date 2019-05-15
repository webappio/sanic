from datetime import datetime

from flask import Flask
from redis import Redis

app = Flask(__name__)
redis = Redis(host='redis', retry_on_timeout=True)


@app.route('/timestamp', methods=['POST'])
def create_timestamp():
    timestamp = datetime.now().isoformat()
    timestamp_id = redis.incr('TIMESTAMP_ID')
    redis.set('TIMESTAMP_{}'.format(timestamp_id), repr(timestamp))
    return {'timestamp': timestamp, 'id': timestamp_id}


@app.route('/timestamp/<id>', methods=['GET'])
def get_timestamp(id):
    return {'timestamp': redis.get('TIMESTAMP_{}'.format(id))}


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=80)
