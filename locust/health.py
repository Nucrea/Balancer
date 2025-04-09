import time
from locust import FastHttpUser, task

class Health(FastHttpUser):
    @task
    def hello_world(self):
        resp = self.client.get("/health")
        assert resp.status_code == 200