import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminDashboardPage } from "@/features/admin/components/sections/dashboard/admin-dashboard";

export default function Page() {
  return (
    <AdminShell activeSection="dashboard" basePath="/admin">
      <AdminDashboardPage />
    </AdminShell>
  );
}
