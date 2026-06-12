import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminModerationPage } from "@/features/admin/components/sections/moderation/admin-moderation";

export default function AdminModerationRoute() {
  return (
    <AdminShell activeSection="moderation" basePath="/admin">
      <AdminModerationPage />
    </AdminShell>
  );
}
