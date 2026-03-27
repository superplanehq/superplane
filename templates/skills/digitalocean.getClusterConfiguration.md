# DigitalOcean Get Cluster Configuration

Use this component to retrieve the active configuration for a DigitalOcean Managed Database cluster.

## Inputs

- `Database Cluster`: The DigitalOcean Managed Database cluster ID

## Output

Returns:

- the cluster ID and cluster name
- the raw `config` object returned by the DigitalOcean API

## Notes

- Requires a DigitalOcean token with `database:read`
- The configuration keys vary by database engine
