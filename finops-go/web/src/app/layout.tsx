import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "FinOps Claw Gang",
  description: "Deterministic FinOps anomaly desk",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-white text-gray-900 font-sans">
        <header className="border-b px-6 py-3">
          <h1 className="text-xl font-bold">FinOps Claw Gang</h1>
        </header>
        <main className="max-w-4xl mx-auto px-6 py-6">{children}</main>
      </body>
    </html>
  );
}
