package models

// MaximumSplitItemsCountOfTransaction represents the maximum count of new transactions created by splitting one transaction
const MaximumSplitItemsCountOfTransaction = 20

// TransactionSplitExpenseItemRequest represents a part of the transaction amount which would be saved as a new expense transaction
type TransactionSplitExpenseItemRequest struct {
	CategoryId int64   `json:"categoryId,string"`
	Amount     int64   `json:"amount"`
	Comment    *string `json:"comment"`
}

// TransactionSplitTransferItemRequest represents a part of the transaction amount which would be saved as a new transfer transaction,
// e.g. the amount paid on behalf of others which is transferred to a receivables account
type TransactionSplitTransferItemRequest struct {
	CategoryId           int64   `json:"categoryId,string"`
	DestinationAccountId int64   `json:"destinationAccountId,string"`
	Amount               int64   `json:"amount"`
	DestinationAmount    int64   `json:"destinationAmount"`
	Comment              *string `json:"comment"`
}

// TransactionSplitRequest represents all parameters of transaction splitting request
type TransactionSplitRequest struct {
	Id            int64                                  `json:"id,string" binding:"required,min=1"`
	CategoryId    int64                                  `json:"categoryId,string" binding:"required,min=1"`
	Amount        int64                                  `json:"amount" binding:"validTransactionAmount"`
	Comment       *string                                `json:"comment" binding:"omitempty,max=255"`
	ExpenseItems  []*TransactionSplitExpenseItemRequest  `json:"expenseItems" binding:"omitempty"`
	TransferItems []*TransactionSplitTransferItemRequest `json:"transferItems" binding:"omitempty"`
}

// TransactionSplitResponse represents the result of transaction splitting
type TransactionSplitResponse struct {
	Transaction     *TransactionInfoResponse   `json:"transaction"`
	NewTransactions []*TransactionInfoResponse `json:"newTransactions"`
}
