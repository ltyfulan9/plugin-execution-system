from sdk.python.pes_client import Client

client = Client(token="demo-token")
print(client.health())
plugins = client.plugins()["data"]["items"]
print("plugins:", [p["name"] for p in plugins])
