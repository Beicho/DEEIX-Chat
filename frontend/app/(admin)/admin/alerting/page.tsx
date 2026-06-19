import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminAlertingPage } from "@/features/admin/components/sections/alerting/admin-alerting-page";

export default function Page() {
  return (
    <AdminShell activeSection="alerting" basePath="/admin">
      <AdminAlertingPage />
    </AdminShell>
  );
}
