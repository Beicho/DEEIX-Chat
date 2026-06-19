import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminArenaPage } from "@/features/admin/components/sections/arena/admin-arena-page";

export default function Page() {
  return (
    <AdminShell activeSection="arena" basePath="/admin">
      <AdminArenaPage />
    </AdminShell>
  );
}
