import { AdminLayout } from "@/components/admin/admin-layout";

export const metadata = { title: "Admin Panel" };

export default function AdminRootLayout({ children }: { children: React.ReactNode }) {
  return <AdminLayout>{children}</AdminLayout>;
}
