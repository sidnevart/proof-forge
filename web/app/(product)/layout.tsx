import type { ReactNode } from "react";

import { ProductNav } from "@/components/product/product-nav";
import { BottomNav } from "@/components/product/bottom-nav";

export default function ProductLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <div className="page-shell">
        <ProductNav />
      </div>
      <main className="product-content">
        {children}
      </main>
      <BottomNav />
    </>
  );
}
