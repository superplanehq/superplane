# digitalocean.deleteDatabase

Delete an existing managed database from a DigitalOcean Managed Database cluster.

## When to use

- Tear down temporary databases after preview or test workflows
- Remove tenant-specific databases during deprovisioning

## Expected inputs

- `Database Cluster`: The cluster containing the database
- `Database`: The database to delete

## Output

- The deleted database name
- The database cluster ID and cluster name
- A `deleted` confirmation flag
