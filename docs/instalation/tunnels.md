When running SuperPlane locally, you won't have a reachable URL to use when configuring event sources, or when pushing outputs from your execution. To address that limitation, you can use [smee.io](https://smee.io/).

### Steps

1. Go to https://smee.io/ and create new channel
2. Install smee-client and start channel pointing to your local superplane

```bash
npm install --global smee-client
```

3. Start forwarding requests from your new channel to the local endpoint you want:

```bash
# Forwarding GH webhook events
smee \
  -p 8000 \
  -P /api/v1/sources/{your_source_id}/github \
  -u https://smee.io/{your_channel_id}

# Forwarding execution outputs
smee \
  -p 8000 \
  -P /api/v1/outputs \
  -u https://smee.io/{your_channel_id}
```
