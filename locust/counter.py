import json
import time
from locust import FastHttpUser, task

class Counter(FastHttpUser):
    @task
    def hello_world(self):
        resp = self.client.get("/count")
        if resp.status_code != 200:
            raise AssertionError('status code not 200, but {0}', resp.status_code)
        if 'count' not in resp.json():
            raise AssertionError('no count field in response json')