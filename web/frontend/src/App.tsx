import { Route, Routes } from "react-router-dom";

import { Navbar } from "@/components/Navbar";
import { HomePage } from "@/pages/Home";
import { SetupPage } from "@/pages/Setup";
import { DocsListPage } from "@/pages/DocsList";
import { DocPage } from "@/pages/Doc";
import { NotFoundPage } from "@/pages/NotFound";

export default function App() {
  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      <main>
        <Routes>
          <Route path="/" element={<HomePage />} />
          {/* Legacy route from the template era. */}
          <Route path="/homepage" element={<HomePage />} />
          <Route path="/setup" element={<SetupPage />} />
          <Route path="/docs" element={<DocsListPage />} />
          <Route path="/docs/:filename" element={<DocPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </main>
    </div>
  );
}
