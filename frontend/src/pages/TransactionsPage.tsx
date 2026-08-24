import { useEffect, useState } from "react";
import { client, getApiErrorMessage } from "@/api/client";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formatMoney } from "@/lib/format";
import type { Account } from "@/types";

type TabKey = "deposit" | "withdraw" | "transfer";

const TABS: { value: TabKey; label: string; action: string }[] = [
  { value: "deposit", label: "Depósito", action: "Depositar" },
  { value: "withdraw", label: "Retiro", action: "Retirar" },
  { value: "transfer", label: "Transferencia", action: "Transferir" },
];

export default function TransactionsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [tab, setTab] = useState<TabKey>("deposit");
  const [accountId, setAccountId] = useState("");
  const [amount, setAmount] = useState("");
  const [toAccount, setToAccount] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    client
      .get<Account[]>("/accounts")
      .then((res) => {
        setAccounts(res.data);
        if (res.data.length) setAccountId(res.data[0].id);
      })
      .catch((e) =>
        toast.error(getApiErrorMessage(e, "No se pudieron cargar las cuentas"))
      );
  }, []);

  const amountNum = Number(amount);
  const validAmount = !isNaN(amountNum) && amountNum > 0;
  const validAccount = !!accountId;
  const validTransfer = tab !== "transfer" || toAccount.trim().length > 0;
  const canSubmit = validAmount && validAccount && validTransfer;

  const selectedAccount = accounts.find((a) => a.id === accountId);
  const accountNumber = selectedAccount?.account_number || accountId;
  const activeTab = TABS.find((t) => t.value === tab)!;

  const confirmText =
    tab === "deposit"
      ? `Confirmar depósito de ${formatMoney(amount)} a la cuenta ${accountNumber}?`
      : tab === "withdraw"
      ? `Confirmar retiro de ${formatMoney(amount)} de la cuenta ${accountNumber}?`
      : `Confirmar transferencia de ${formatMoney(amount)} de la cuenta ${accountNumber} a la cuenta ${toAccount}?`;

  const resetForm = () => {
    setAmount("");
    setToAccount("");
    setConfirmOpen(false);
  };

  const execute = async () => {
    setSubmitting(true);
    try {
      if (tab === "deposit") {
        await client.post("/transactions/deposit", {
          account_id: accountId,
          amount,
        });
      } else if (tab === "withdraw") {
        await client.post("/transactions/withdraw", {
          account_id: accountId,
          amount,
        });
      } else {
        await client.post("/transactions/transfer", {
          from_account: accountId,
          to_account: toAccount,
          amount,
        });
      }
      toast.success(`Operación de ${activeTab.label} realizada con éxito`);
      resetForm();
    } catch (err) {
      toast.error(
        getApiErrorMessage(err, "No se pudo completar la operación")
      );
      setConfirmOpen(false);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Transacciones</h1>
        <p className="text-sm text-muted-foreground">
          Realiza depósitos, retiros y transferencias.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Nueva operación</CardTitle>
          <CardDescription>
            Selecciona el tipo de operación y completa el formulario.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs
            value={tab}
            onValueChange={(v) => {
              setTab(v as TabKey);
              resetForm();
            }}
          >
            <TabsList className="grid w-full grid-cols-3">
              {TABS.map((t) => (
                <TabsTrigger key={t.value} value={t.value}>
                  {t.label}
                </TabsTrigger>
              ))}
            </TabsList>
            <TabsContent value={tab} className="space-y-4 pt-2">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Cuenta origen</label>
                <Select
                  value={accountId}
                  onChange={(e) => setAccountId(e.target.value)}
                >
                  <option value="">Selecciona una cuenta</option>
                  {accounts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.account_number} — {a.account_type} (
                      {formatMoney(a.balance, a.currency)})
                    </option>
                  ))}
                </Select>
              </div>

              {tab === "transfer" && (
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Cuenta destino</label>
                  <Input
                    value={toAccount}
                    onChange={(e) => setToAccount(e.target.value)}
                    placeholder="Número de cuenta destino"
                  />
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-sm font-medium">Monto</label>
                <Input
                  type="number"
                  min="0"
                  step="0.01"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="0.00"
                />
              </div>

              <Button
                className="w-full"
                disabled={!canSubmit}
                onClick={() => setConfirmOpen(true)}
              >
                {activeTab.action}
              </Button>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirmar operación</DialogTitle>
            <DialogDescription>{confirmText}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmOpen(false)}
              disabled={submitting}
            >
              Cancelar
            </Button>
            <Button onClick={execute} disabled={submitting}>
              {submitting && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Confirmar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
