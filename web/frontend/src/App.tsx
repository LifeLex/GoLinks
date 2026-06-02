import type { ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";

import { Navbar } from "@/components/Navbar";
import { HomePage } from "@/pages/Home";
import { SetupPage } from "@/pages/Setup";
import { DocsListPage } from "@/pages/DocsList";
import { DocPage } from "@/pages/Doc";
import { NotFoundPage } from "@/pages/NotFound";
import { LoginPage } from "@/pages/Login";
import { WelcomePage } from "@/pages/Welcome";
import { AdminUsersPage } from "@/pages/AdminUsers";
import { useAuth } from "@/auth/AuthContext";

/** Restricts a route to admins, redirecting others to login or home. */
function RequireAdmin({ children }: { children: ReactNode }) {
  const { isLoading, isAuthenticated, isAdmin } = useAuth();
  if (isLoading) return null;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (!isAdmin) return <Navigate to="/" replace />;
  return <>{children}</>;
}

export default function App() {
  const { isLoading, needsSetup } = useAuth();
  const location = useLocation();

  // Hold rendering until we know the auth state, so we don't flash the wrong UI.
  if (isLoading) return null;

  // On a fresh instance, force everyone to the first-run wizard.
  if (needsSetup && location.pathname !== "/welcome") {
    return <Navigate to="/welcome" replace />;
  }
  // Once set up, the wizard is no longer reachable.
  if (!needsSetup && location.pathname === "/welcome") {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      <main>
        <Routes>
          <Route path="/" element={<HomePage />} />
          {/* Legacy route from the template era. */}
          <Route path="/homepage" element={<HomePage />} />
          <Route path="/setup" element={<SetupPage />} />
          <Route path="/welcome" element={<WelcomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/docs" element={<DocsListPage />} />
          <Route path="/docs/:filename" element={<DocPage />} />
          <Route
            path="/admin/users"
            element={
              <RequireAdmin>
                <AdminUsersPage />
              </RequireAdmin>
            }
          />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </main>
    </div>
  );
}
