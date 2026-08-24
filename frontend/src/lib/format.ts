export function formatMoney(amount: string | number, currency = "USD"): string {
  const num = typeof amount === "string" ? parseFloat(amount) : amount;
  if (isNaN(num)) return "$0.00";
  const sign = num < 0 ? "-" : "";
  const formatted = Math.abs(num).toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  const symbol = currency === "USD" ? "$" : currency + " ";
  return `${sign}${symbol}${formatted}`;
}

export function formatDate(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const date = d.toLocaleDateString("es-HN", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
  const time = d.toLocaleTimeString("es-HN", {
    hour: "2-digit",
    minute: "2-digit",
  });
  return `${date} ${time}`;
}

export function amountStyle(type: string): { sign: string; className: string } {
  if (type === "deposit") return { sign: "+", className: "text-green-600" };
  if (type === "withdraw") return { sign: "-", className: "text-red-600" };
  return { sign: "", className: "text-foreground" };
}

export function typeLabel(type: string): string {
  switch (type) {
    case "deposit":
      return "Depósito";
    case "withdraw":
      return "Retiro";
    case "transfer":
      return "Transferencia";
    default:
      return type;
  }
}
