# digitalocean.createDatabaseCluster

Create a new DigitalOcean Managed Database cluster.

## When to use

- Provision a fresh managed database cluster for a service or environment
- Create database infrastructure before downstream app, database, or migration steps

## Expected inputs

- `Name`
- `Engine`
- `Version`
- `Region`
- `Size`
- `Node Count`

## Output

- The created cluster object, including its ID, engine, version, region, size, and status
