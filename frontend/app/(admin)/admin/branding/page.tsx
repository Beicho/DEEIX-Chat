import { AdminShell } from "@/features/admin/components/admin-shell";
import { AdminBrandingSettingsPage } from "@/features/admin/components/sections/branding/admin-branding";

export default function Page() {
  return (
    <AdminShell activeSection="branding" basePath="/admin">
      <AdminBrandingSettingsPage />
    </AdminShell>
  );
}
