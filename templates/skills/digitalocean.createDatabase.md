# digitalocean.createDatabase

Create a new managed database inside an existing DigitalOcean Managed Database cluster.

## When to use

- Provision an application-specific database during environment setup
- Create a tenant database before running migrations or imports

## Expected inputs

- `Database Cluster`: The target managed database cluster
- `Database Name`: The database name to create

## Output

- The created database name
- The database cluster ID and cluster name
