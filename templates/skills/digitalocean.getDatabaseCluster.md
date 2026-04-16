# digitalocean.getDatabaseCluster

Retrieve details of an existing DigitalOcean Managed Database cluster.

## When to use

- Inspect a cluster before creating databases, users, or connection pools
- Read connection details, sizing, and current status for routing or validation steps

## Expected inputs

- `Database Cluster`

## Output

- The database cluster object, including ID, engine, version, region, size, node count, status, and connection details
