export type Account = {
  id: string;
  account_number: string;
  account_type: string;
  currency: string;
  balance: string;
  created_at?: string;
};

export type TransactionType = "deposit" | "withdraw" | "transfer";

export type Transaction = {
  id: string;
  from_account: string;
  to_account: string;
  amount: string;
  type: TransactionType | string;
  description: string;
  timestamp: string;
  status: string;
};

export type TransactionListResponse = {
  transactions: Transaction[];
};

export type ChatMessage = {
  role: "user" | "assistant" | "system";
  content: string;
};

export type PendingAction = Record<string, string | number | boolean | null> | null;

export type ChatResponse = {
  message: string;
  requires_confirmation?: boolean;
  pending_action?: PendingAction;
};
