import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { formatDate, amountStyle, formatMoney, typeLabel } from "@/lib/format";
import type { Transaction } from "@/types";

interface TransactionTableProps {
  transactions: Transaction[];
  emptyText?: string;
}

export function TransactionTable({ transactions, emptyText = "Sin transacciones" }: TransactionTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Fecha</TableHead>
          <TableHead>Tipo</TableHead>
          <TableHead>De</TableHead>
          <TableHead>Para</TableHead>
          <TableHead className="text-right">Monto</TableHead>
          <TableHead>Descripción</TableHead>
          <TableHead>Estado</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {transactions.length === 0 ? (
          <TableRow>
            <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
              {emptyText}
            </TableCell>
          </TableRow>
        ) : (
          transactions.map((t) => {
            const { sign, className } = amountStyle(t.type);
            return (
              <TableRow key={t.id}>
                <TableCell className="whitespace-nowrap">{formatDate(t.timestamp)}</TableCell>
                <TableCell>
                  <Badge variant="outline">{typeLabel(t.type)}</Badge>
                </TableCell>
                <TableCell>{t.from_account || "—"}</TableCell>
                <TableCell>{t.to_account || "—"}</TableCell>
                <TableCell className={`text-right font-medium ${className}`}>
                  {sign}
                  {formatMoney(t.amount)}
                </TableCell>
                <TableCell>{t.description || "—"}</TableCell>
                <TableCell>
                  <Badge variant="secondary">{t.status || "completed"}</Badge>
                </TableCell>
              </TableRow>
            );
          })
        )}
      </TableBody>
    </Table>
  );
}
