import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Landmark, Loader2 } from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { getApiErrorMessage } from "@/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

export default function AuthPage() {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [error, setError] = useState("");
  const { login, register, loading } = useAuth();
  const navigate = useNavigate();

  const validate = () => {
    if (!email.trim() || !password) {
      setError("Email y contraseña son obligatorios.");
      return false;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setError("Ingresa un email válido.");
      return false;
    }
    if (mode === "register" && fullName.trim().length < 2) {
      setError("El nombre completo es obligatorio.");
      return false;
    }
    setError("");
    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;
    try {
      if (mode === "login") {
        await login(email, password);
        toast.success("Bienvenido a HNL Bank");
      } else {
        await register(email, password, fullName);
        toast.success("Cuenta creada correctamente.");
      }
      navigate("/dashboard", { replace: true });
    } catch (err) {
      const msg = getApiErrorMessage(
        err,
        mode === "login" ? "Credenciales incorrectas" : "No se pudo registrar"
      );
      setError(msg);
      toast.error(msg);
    }
  };

  const switchMode = (m: "login" | "register") => {
    setMode(m);
    setError("");
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
            <Landmark className="h-6 w-6" />
          </div>
          <CardTitle className="text-2xl">HNL Bank</CardTitle>
          <CardDescription>
            {mode === "login"
              ? "Inicia sesión en tu cuenta"
              : "Crea una cuenta nueva"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex rounded-lg bg-muted p-1">
            <button
              type="button"
              onClick={() => switchMode("login")}
              className={cn(
                "flex-1 rounded-md py-1.5 text-sm font-medium",
                mode === "login"
                  ? "bg-background text-foreground shadow"
                  : "text-muted-foreground"
              )}
            >
              Iniciar sesión
            </button>
            <button
              type="button"
              onClick={() => switchMode("register")}
              className={cn(
                "flex-1 rounded-md py-1.5 text-sm font-medium",
                mode === "register"
                  ? "bg-background text-foreground shadow"
                  : "text-muted-foreground"
              )}
            >
              Registrarse
            </button>
          </div>

          <form onSubmit={onSubmit} className="space-y-3">
            {mode === "register" && (
              <Input
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Nombre completo"
                autoComplete="name"
              />
            )}
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="Email"
              autoComplete="email"
            />
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Contraseña"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
            />
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="h-4 w-4 animate-spin" />}
              {mode === "login" ? "Iniciar sesión" : "Crear cuenta"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
