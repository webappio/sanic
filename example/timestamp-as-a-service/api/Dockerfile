FROM ubuntu:18.04

RUN apt-get update && apt-get install -y python3-pip python3-dev && rm -rf /var/lib/apt/lists/*
ADD requirements.txt /tmp
RUN python3 -m pip install -r /tmp/requirements.txt && rm /tmp/requirements.txt

WORKDIR /lib/app/

ADD app.py ./
ENTRYPOINT ["python3", "app.py"]