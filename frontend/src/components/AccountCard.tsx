import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatMoney } from "@/lib/format";
import type { Account } from "@/types";

export function AccountCard({ account }: { account: Account }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium capitalize">
          {account.account_type}
        </CardTitle>
        <Badge variant="secondary">{account.account_type}</Badge>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {formatMoney(account.balance, account.currency)}
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          Cuenta {account.account_number}
        </p>
      </CardContent>
    </Card>
  );
}
