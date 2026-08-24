import { useCallback, useEffect, useState } from "react";
import { client, getApiErrorMessage } from "@/api/client";
import { useAuth } from "@/context/AuthContext";
import { AccountCard } from "@/components/AccountCard";
import { TransactionTable } from "@/components/TransactionTable";
import { ChatPanel } from "@/components/ChatPanel";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import type { Account, Transaction, TransactionListResponse } from "@/types";

function dedupeAndSort(list: Transaction[]): Transaction[] {
  const map = new Map<string, Transaction>();
  for (const t of list) {
    if (t.id) map.set(t.id, t);
  }
  return Array.from(map.values()).sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );
}

export default function DashboardPage() {
  const { user } = useAuth();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const accRes = await client.get<Account[]>("/accounts");
      const accs = accRes.data;
      setAccounts(accs);
      const lists = await Promise.all(
        accs.map(async (a) => {
          const res = await client.get<TransactionListResponse>(
            `/transactions/${a.id}?limit=10&offset=0`
          );
          return res.data.transactions;
        })
      );
      setTransactions(dedupeAndSort(lists.flat()).slice(0, 10));
    } catch (e) {
      toast.error(getApiErrorMessage(e, "No se pudieron cargar los datos"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const totalBalance = accounts.reduce(
    (sum, a) => sum + (parseFloat(a.balance) || 0),
    0
  );

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">
          Hola, {user?.full_name || "Usuario"}
        </h1>
        <p className="text-sm text-muted-foreground">
          Resumen de tus cuentas
        </p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Saldo total</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold">
            {formatMoney(totalBalance.toFixed(2))}
          </div>
        </CardContent>
      </Card>

      <div>
        <h2 className="mb-2 text-lg font-semibold">Cuentas</h2>
        {loading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Cargando…
          </div>
        ) : accounts.length === 0 ? (
          <p className="text-muted-foreground">No tienes cuentas registradas.</p>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {accounts.map((a) => (
              <AccountCard key={a.id} account={a} />
            ))}
          </div>
        )}
      </div>

      <div>
        <h2 className="mb-2 text-lg font-semibold">Últimas transacciones</h2>
        <Card>
          <CardContent className="p-0">
            <TransactionTable
              transactions={transactions}
              emptyText="Aún no hay transacciones"
            />
          </CardContent>
        </Card>
      </div>

      <ChatPanel onActionComplete={loadData} />
    </div>
  );
}
