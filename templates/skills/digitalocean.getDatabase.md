# DigitalOcean Get Database

Use this component to retrieve a managed database from a DigitalOcean cluster and enrich it with cluster context.

## Inputs

- `Database Cluster`: The DigitalOcean Managed Database cluster ID
- `Database`: The database name inside that cluster

## Output

Returns:

- the database name
- the cluster ID and cluster name
- cluster engine, version, region, and status
- cluster connection details when available
- the raw database object returned by the DigitalOcean API

## Notes

- Requires a DigitalOcean token with `database:read`
- Database management is not supported for Caching or Valkey clusters
