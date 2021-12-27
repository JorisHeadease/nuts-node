package events

// PrivateTransactionsSubject defines the NATS subject used for private transactions in the v2 protocol
//
// Payload: dag.Transaction
const PrivateTransactionsSubject = "nuts.v2.private-transactions"
