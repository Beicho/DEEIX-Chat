import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminCheckInPage } from "@/features/admin/components/sections/checkin/admin-checkin-page";

export default function Page() {
  return (
    <AdminShell activeSection="checkin" basePath="/admin">
      <AdminCheckInPage />
    </AdminShell>
  );
}
