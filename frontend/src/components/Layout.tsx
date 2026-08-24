import { useState } from "react";
import { Link, NavLink, useNavigate, Outlet } from "react-router-dom";
import {
  LayoutDashboard,
  ArrowLeftRight,
  History,
  LogOut,
  Menu,
  X,
  Landmark,
} from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/transactions", label: "Transacciones", icon: ArrowLeftRight },
  { to: "/history", label: "Historial", icon: History },
];

export function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate("/auth");
  };

  const displayName = user?.full_name || user?.email || "Usuario";

  const NavLinks = ({ className }: { className?: string }) => (
    <nav className={cn("flex flex-col gap-1", className)}>
      {navItems.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          onClick={() => setOpen(false)}
          className={({ isActive }) =>
            cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            )
          }
        >
          <item.icon className="h-4 w-4" />
          {item.label}
        </NavLink>
      ))}
    </nav>
  );

  return (
    <div className="min-h-screen bg-background">
      {/* Mobile top bar */}
      <header className="flex items-center justify-between border-b px-4 py-3 lg:hidden">
        <Link to="/dashboard" className="flex items-center gap-2 font-semibold">
          <Landmark className="h-5 w-5" />
          HNL Bank
        </Link>
        <Button variant="ghost" size="icon" onClick={() => setOpen((o) => !o)}>
          {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </Button>
      </header>

      {/* Mobile drawer */}
      {open && (
        <div className="border-b bg-card px-4 py-3 lg:hidden">
          <NavLinks />
        </div>
      )}

      <div className="flex">
        {/* Desktop sidebar */}
        <aside className="hidden w-64 shrink-0 flex-col border-r bg-card lg:flex">
          <div className="flex items-center gap-2 px-4 py-4 font-semibold">
            <Landmark className="h-5 w-5" />
            HNL Bank
          </div>
          <NavLinks className="px-3" />
          <div className="mt-auto border-t p-4">
            <div className="mb-2 truncate text-sm font-medium">{displayName}</div>
            <Button variant="outline" className="w-full" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
              Cerrar sesión
            </Button>
          </div>
        </aside>

        <main className="flex-1 p-4 md:p-6">
          {/* Mobile user + logout */}
          <div className="mb-4 flex items-center justify-between lg:hidden">
            <span className="truncate text-sm text-muted-foreground">{displayName}</span>
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
              Salir
            </Button>
          </div>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
