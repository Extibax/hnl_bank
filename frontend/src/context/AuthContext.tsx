import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import { client, TOKEN_KEY, USER_KEY } from "@/api/client";

export type User = {
  id: string;
  email: string;
  full_name: string;
  created_at?: string;
};

type AuthContextValue = {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<User>;
  register: (
    email: string,
    password: string,
    full_name: string
  ) => Promise<User>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function readUser(): User | null {
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => readUser());
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem(TOKEN_KEY)
  );
  const [loading, setLoading] = useState(false);

  const persist = useCallback((t: string, u: User) => {
    localStorage.setItem(TOKEN_KEY, t);
    localStorage.setItem(USER_KEY, JSON.stringify(u));
    setToken(t);
    setUser(u);
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      setLoading(true);
      try {
        const res = await client.post<{ token: string; user?: User }>("/auth/login", {
          email,
          password,
        });
        const t = res.data.token;
        const u: User =
          res.data.user ??
          (() => {
            const derivedName = email.includes("@")
              ? email.split("@")[0].replace(/[._-]+/g, " ")
              : email;
            return {
              id: "",
              email,
              full_name:
                derivedName.charAt(0).toUpperCase() + derivedName.slice(1),
            };
          })();
        persist(t, u);
        return u;
      } finally {
        setLoading(false);
      }
    },
    [persist]
  );

  const register = useCallback(
    async (email: string, password: string, full_name: string) => {
      setLoading(true);
      try {
        const res = await client.post<User>("/auth/register", {
          email,
          password,
          full_name,
        });
        const regUser = res.data;
        await login(email, password);
        const freshToken = localStorage.getItem(TOKEN_KEY) || "";
        persist(freshToken, regUser);
        return regUser;
      } finally {
        setLoading(false);
      }
    },
    [login, persist, token]
  );

  const logout = useCallback(async () => {
    try {
      await client.post("/auth/logout");
    } catch {
      // ignore network errors during logout
    }
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    setToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{ user, token, loading, login, register, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth debe usarse dentro de AuthProvider");
  return ctx;
}