import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminSecurityPage } from "@/features/admin/components/sections/security/admin-security";

export default function Page() {
  return (
    <AdminShell activeSection="security" basePath="/admin">
      <AdminSecurityPage />
    </AdminShell>
  );
}
