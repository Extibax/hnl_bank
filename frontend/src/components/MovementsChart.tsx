import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { formatMoney } from "@/lib/format";
import type { Transaction } from "@/types";

interface MovementsChartProps {
  transactions: Transaction[];
}

type TypeKey = "deposit" | "withdraw" | "transfer";

const BARS: { key: TypeKey; label: string; fill: string }[] = [
  { key: "deposit", label: "Depósitos", fill: "#16a34a" },
  { key: "withdraw", label: "Retiros", fill: "#dc2626" },
  { key: "transfer", label: "Transferencias", fill: "#6b7280" },
];

function compactMoney(v: number): string {
  const abs = Math.abs(v);
  const sign = v < 0 ? "-" : "";
  if (abs >= 1_000_000) return `${sign}$${(abs / 1_000_000).toFixed(1)}M`;
  if (abs >= 1_000) return `${sign}$${(abs / 1_000).toFixed(1)}k`;
  return `${sign}$${abs.toFixed(0)}`;
}

export function MovementsChart({ transactions }: MovementsChartProps) {
  const totals: Record<TypeKey, number> = { deposit: 0, withdraw: 0, transfer: 0 };

  for (const t of transactions) {
    const amount = parseFloat(t.amount) || 0;
    if (t.type === "deposit") totals.deposit += amount;
    else if (t.type === "withdraw") totals.withdraw += amount;
    else if (t.type === "transfer") totals.transfer += amount;
  }

  const data = BARS.map((b) => ({
    name: b.label,
    total: totals[b.key],
    fill: b.fill,
  }));

  return (
    <div className="w-full">
      <div className="h-56 w-full sm:h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 8 }}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="name"
              tickLine={false}
              axisLine={false}
              tick={{ fontSize: 12 }}
            />
            <YAxis
              tickFormatter={(v: number) => compactMoney(v)}
              tickLine={false}
              axisLine={false}
              width={64}
            />
            <Tooltip
              formatter={(value) => formatMoney(Number(value))}
              cursor={{ fill: "rgba(0,0,0,0.04)" }}
            />
            <Bar dataKey="total" radius={[4, 4, 0, 0]}>
              {data.map((entry) => (
                <Cell key={entry.name} fill={entry.fill} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-3 flex flex-wrap items-center justify-center gap-4">
        {BARS.map((b) => (
          <span
            key={b.key}
            className="flex items-center gap-1.5 text-xs text-muted-foreground"
          >
            <span
              className="h-3 w-3 rounded-full"
              style={{ backgroundColor: b.fill }}
            />
            {b.label}
          </span>
        ))}
      </div>
    </div>
  );
}
