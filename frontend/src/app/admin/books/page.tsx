"use client";

import { useState, useEffect } from "react";
import {
  listBooks,
  createBook,
  deleteBook,
  processBook,
} from "@/lib/api";
import type { Book } from "@/lib/types";
import Pagination from "@/components/Pagination";

export default function AdminBooksPage() {
  const [books, setBooks] = useState<Book[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    sourceUrl: "",
    sourceType: "url",
    subject: "",
    gradeLevels: "",
    title: "",
    author: "",
  });

  useEffect(() => {
    loadBooks();
  }, [page]);

  async function loadBooks() {
    setLoading(true);
    try {
      const res = await listBooks({ page, limit: 20 });
      setBooks(res.data || []);
      setTotalPages(res.pagination.total_pages);
    } catch {
      setBooks([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createBook({
        sourceUrl: form.sourceUrl,
        sourceType: form.sourceType,
        subject: form.subject,
        gradeLevels: form.gradeLevels.split(",").map((g) => g.trim()),
        title: form.title || undefined,
        author: form.author || undefined,
      });
      setShowForm(false);
      setForm({ sourceUrl: "", sourceType: "url", subject: "", gradeLevels: "", title: "", author: "" });
      loadBooks();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create book");
    }
  }

  async function handleProcess(id: string) {
    try {
      await processBook(id);
      loadBooks();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to start processing");
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this book and all associated data?")) return;
    try {
      await deleteBook(id);
      loadBooks();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete book");
    }
  }

  const statusColors: Record<string, string> = {
    ready: "bg-green-100 text-green-800",
    processing: "bg-yellow-100 text-yellow-800",
    pending: "bg-gray-100 text-gray-800",
    failed: "bg-red-100 text-red-800",
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Manage Books</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-blue-700"
        >
          {showForm ? "Cancel" : "Add Book from URL"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white rounded-lg shadow p-4 mb-6 space-y-3">
          <input
            placeholder="Source URL"
            value={form.sourceUrl}
            onChange={(e) => setForm({ ...form, sourceUrl: e.target.value })}
            required
            className="w-full border rounded px-3 py-2 text-sm"
          />
          <div className="grid grid-cols-2 gap-3">
            <input
              placeholder="Subject *"
              value={form.subject}
              onChange={(e) => setForm({ ...form, subject: e.target.value })}
              required
              className="border rounded px-3 py-2 text-sm"
            />
            <input
              placeholder="Grade Levels (comma-separated) *"
              value={form.gradeLevels}
              onChange={(e) => setForm({ ...form, gradeLevels: e.target.value })}
              required
              className="border rounded px-3 py-2 text-sm"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <input
              placeholder="Title (optional)"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              className="border rounded px-3 py-2 text-sm"
            />
            <input
              placeholder="Author (optional)"
              value={form.author}
              onChange={(e) => setForm({ ...form, author: e.target.value })}
              className="border rounded px-3 py-2 text-sm"
            />
          </div>
          <button type="submit" className="bg-green-600 text-white px-4 py-2 rounded text-sm hover:bg-green-700">
            Create Book
          </button>
        </form>
      )}

      {loading ? (
        <div className="text-center py-12">Loading...</div>
      ) : (
        <div className="space-y-3">
          {books.map((book) => (
            <div key={book.id} className="bg-white rounded-lg shadow p-4 flex justify-between items-center">
              <div>
                <h3 className="font-medium">{book.title}</h3>
                <p className="text-sm text-gray-500">{book.author} &middot; {book.subject}</p>
                <div className="flex gap-2 mt-1">
                  <span className={`text-xs px-2 py-0.5 rounded ${statusColors[book.status] || "bg-gray-100"}`}>
                    {book.status}
                  </span>
                  <span className="text-xs text-gray-400">{book.sourceType}</span>
                </div>
              </div>
              <div className="flex gap-2">
                {(book.status === "pending" || book.status === "failed") && (
                  <button
                    onClick={() => handleProcess(book.id)}
                    className="text-blue-600 text-sm hover:underline"
                  >
                    Process
                  </button>
                )}
                <button
                  onClick={() => handleDelete(book.id)}
                  className="text-red-600 text-sm hover:underline"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-6">
          <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
        </div>
      )}
    </div>
  );
}
