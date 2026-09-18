package workers

// workerPollBatchSize bounds how many eligible rows a worker reads per poll.
// It matches other sweep limits (pending runs, started runs, repository
// provisioner) and exceeds the typical semaphore weight of 25 so a replica
// can keep slots filled without loading the full backlog.
const workerPollBatchSize = 100
