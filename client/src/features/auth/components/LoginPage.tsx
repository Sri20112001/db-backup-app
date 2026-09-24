import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authApi, orgApi } from "@/services/api";
import { useAuthStore } from "@/store/authStore";
import { useUIStore } from "@/store/uiStore";
import { Shield, Eye, EyeOff, Loader2 } from "lucide-react";

const LoginPage = () => {
  const navigate = useNavigate();
  const { setAuth, setCurrentOrg } = useAuthStore();
  const { addToast } = useUIStore();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setIsLoading(true);
    try {
      const data = await authApi.login(email, password);
      setAuth(data.user, data.access_token, data.refresh_token);
      const orgs = await orgApi.list();
      if (orgs.length > 0) setCurrentOrg(orgs[0]);
      addToast("success", `Welcome back, ${data.user.name}`);
      navigate(orgs.length > 0 ? "/" : "/setup");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#f9f9ff] flex items-center justify-center px-4">
      <div className="w-full max-w-[420px] bg-[#ffffff] rounded-xl border border-[#e9edff] shadow-[0_4px_24px_rgba(20,27,43,0.08)] p-8">
        {/* Logo */}
        <div className="flex items-center gap-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-[#2563eb] flex items-center justify-center">
            <Shield size={20} className="text-white" />
          </div>
          <div>
            <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">
              VaultGuard
            </h1>
            <p className="text-[12px] text-[#737686]">
              Backup & Recovery Platform
            </p>
          </div>
        </div>

        <h2 className="text-[18px] font-semibold text-[#141b2b] mb-1">
          Sign in
        </h2>
        <p className="text-[13px] text-[#737686] mb-6">
          Protecting your business data
        </p>

        {error && (
          <div className="mb-4 px-3 py-2.5 rounded-lg bg-[#ffdad6] border border-[#fca5a5] text-[#93000a] text-[13px]">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div>
            <label
              className="block text-[13px] font-medium text-[#434655] mb-1.5"
              htmlFor="email"
            >
              Email address
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="admin@company.com"
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] placeholder:text-[#737686] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
          </div>

          <div>
            <label
              className="block text-[13px] font-medium text-[#434655] mb-1.5"
              htmlFor="password"
            >
              Password
            </label>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                placeholder="••••••••"
                className="w-full h-9 px-3 pr-10 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] placeholder:text-[#737686] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#737686] hover:text-[#141b2b]"
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full h-10 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] active:bg-[#1e40af] transition-colors disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {isLoading && <Loader2 size={14} className="animate-spin" />}
            {isLoading ? "Signing in..." : "Sign in"}
          </button>
        </form>

        <p className="mt-6 text-center text-[13px] text-[#737686]">
          Don't have an account?{" "}
          <Link
            to="/register"
            className="text-[#004ac6] hover:underline font-medium"
          >
            Register
          </Link>
        </p>
      </div>
    </div>
  );
};

export default LoginPage;
