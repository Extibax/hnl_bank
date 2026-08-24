import { useEffect, useMemo, useState } from "react";
import { client, getApiErrorMessage } from "@/api/client";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import {
  formatDate,
  amountStyle,
  formatMoney,
  typeLabel,
} from "@/lib/format";
import type { Account, Transaction, TransactionListResponse } from "@/types";

const PER_PAGE = 20;

function matchesAccount(t: Transaction, acc: Account): boolean {
  return (
    t.from_account === acc.account_number ||
    t.to_account === acc.account_number ||
    t.from_account === acc.id ||
    t.to_account === acc.id
  );
}

export default function HistoryPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [all, setAll] = useState<Transaction[]>([]);
  const [filter, setFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);

  useEffect(() => {
    (async () => {
      try {
        const accRes = await client.get<Account[]>("/accounts");
        const accs = accRes.data;
        setAccounts(accs);
        const lists = await Promise.all(
          accs.map(async (a) => {
            const res = await client.get<TransactionListResponse>(
              `/transactions/${a.id}?limit=1000&offset=0`
            );
            return res.data.transactions;
          })
        );
        const map = new Map<string, Transaction>();
        for (const t of lists.flat()) {
          if (t.id) map.set(t.id, t);
        }
        setAll(
          Array.from(map.values()).sort(
            (a, b) =>
              new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
          )
        );
      } catch (e) {
        toast.error(
          getApiErrorMessage(e, "No se pudieron cargar las transacciones")
        );
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const filtered = useMemo(() => {
    if (filter === "all") return all;
    const acc = accounts.find((a) => a.id === filter);
    return acc ? all.filter((t) => matchesAccount(t, acc)) : all;
  }, [all, filter, accounts]);

  useEffect(() => {
    setPage(0);
  }, [filter]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PER_PAGE));
  const currentPage = Math.min(page, totalPages - 1);
  const pageItems = filtered.slice(
    currentPage * PER_PAGE,
    currentPage * PER_PAGE + PER_PAGE
  );

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Historial</h1>
          <p className="text-sm text-muted-foreground">
            Todas tus transacciones.
          </p>
        </div>
        <div className="w-full sm:w-64">
          <Select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            <option value="all">Todas las cuentas</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.account_number}
              </option>
            ))}
          </Select>
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="flex items-center justify-center gap-2 py-10 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Cargando…
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Fecha</TableHead>
                    <TableHead>Tipo</TableHead>
                    <TableHead className="hidden md:table-cell">De</TableHead>
                    <TableHead className="hidden md:table-cell">Para</TableHead>
                    <TableHead className="text-right">Monto</TableHead>
                    <TableHead className="hidden lg:table-cell">Descripción</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={6}
                        className="py-8 text-center text-muted-foreground"
                      >
                        Sin transacciones
                      </TableCell>
                    </TableRow>
                  ) : (
                    pageItems.map((t) => {
                      const { sign, className } = amountStyle(t.type);
                      return (
                        <TableRow key={t.id}>
                          <TableCell className="whitespace-nowrap">
                            {formatDate(t.timestamp)}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{typeLabel(t.type)}</Badge>
                          </TableCell>
                          <TableCell className="hidden md:table-cell">{t.from_account || "—"}</TableCell>
                          <TableCell className="hidden md:table-cell">{t.to_account || "—"}</TableCell>
                          <TableCell
                            className={`text-right font-medium ${className}`}
                          >
                            {sign}
                            {formatMoney(t.amount)}
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">{t.description || "—"}</TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
              <div className="flex items-center justify-between border-t p-4">
                <span className="text-sm text-muted-foreground">
                  {filtered.length} transacciones · Página {currentPage + 1} de{" "}
                  {totalPages}
                </span>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={currentPage === 0}
                    onClick={() => setPage(currentPage - 1)}
                  >
                    Anterior
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={currentPage >= totalPages - 1}
                    onClick={() => setPage(currentPage + 1)}
                  >
                    Siguiente
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}