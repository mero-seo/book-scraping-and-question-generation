"use client";

import { useState, useEffect } from "react";
import { listSources, createSource, deleteSource } from "@/lib/api";
import type { AllowedSource } from "@/lib/types";

export default function AdminSourcesPage() {
  const [sources, setSources] = useState<AllowedSource[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    urlPattern: "",
    name: "",
    sourceType: "scrape",
    notes: "",
  });

  useEffect(() => {
    loadSources();
  }, []);

  async function loadSources() {
    try {
      const res = await listSources({ limit: 100 });
      setSources(res.data || []);
    } catch {
      // Error
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createSource(form);
      setForm({ urlPattern: "", name: "", sourceType: "scrape", notes: "" });
      setShowForm(false);
      loadSources();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create source");
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this source?")) return;
    try {
      await deleteSource(id);
      loadSources();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete source");
    }
  }

  if (loading) return <div className="text-center py-12">Loading...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Allowed Sources</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-blue-700"
        >
          {showForm ? "Cancel" : "Add Source"}
        </button>
      </div>

      {showForm && (
        <form
          onSubmit={handleCreate}
          className="bg-white rounded-lg shadow p-4 mb-6 space-y-3"
        >
          <input
            placeholder="URL Pattern (e.g., https://gutenberg.org/*)"
            value={form.urlPattern}
            onChange={(e) => setForm({ ...form, urlPattern: e.target.value })}
            required
            className="w-full border rounded px-3 py-2 text-sm"
          />
          <input
            placeholder="Name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
            className="w-full border rounded px-3 py-2 text-sm"
          />
          <select
            value={form.sourceType}
            onChange={(e) => setForm({ ...form, sourceType: e.target.value })}
            className="w-full border rounded px-3 py-2 text-sm"
          >
            <option value="scrape">Scrape</option>
            <option value="api">API</option>
          </select>
          <input
            placeholder="Notes (optional)"
            value={form.notes}
            onChange={(e) => setForm({ ...form, notes: e.target.value })}
            className="w-full border rounded px-3 py-2 text-sm"
          />
          <button
            type="submit"
            className="bg-green-600 text-white px-4 py-2 rounded text-sm hover:bg-green-700"
          >
            Save
          </button>
        </form>
      )}

      <div className="space-y-3">
        {sources.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No sources configured</p>
        ) : (
          sources.map((s) => (
            <div
              key={s.id}
              className="bg-white rounded-lg shadow p-4 flex justify-between items-center"
            >
              <div>
                <h3 className="font-medium">{s.name}</h3>
                <p className="text-sm text-gray-500">{s.urlPattern}</p>
                <div className="flex gap-2 mt-1">
                  <span className="text-xs bg-gray-100 px-2 py-0.5 rounded">
                    {s.sourceType}
                  </span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded ${s.enabled ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}
                  >
                    {s.enabled ? "Enabled" : "Disabled"}
                  </span>
                </div>
              </div>
              <button
                onClick={() => handleDelete(s.id)}
                className="text-red-600 text-sm hover:underline"
              >
                Delete
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
