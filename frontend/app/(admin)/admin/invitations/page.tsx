import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminInvitationsPage } from "@/features/admin/components/sections/invitations/admin-invitations";

export default function Page() {
  return (
    <AdminShell activeSection="invitations" basePath="/admin">
      <AdminInvitationsPage />
    </AdminShell>
  );
}
