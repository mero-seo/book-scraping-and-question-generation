"use client";

import { useState, useEffect } from "react";
import { listBooks, searchBooks } from "@/lib/api";
import type { Book, SearchResult } from "@/lib/types";
import BookGrid from "@/components/BookGrid";
import SearchBar from "@/components/SearchBar";
import Pagination from "@/components/Pagination";

export default function HomePage() {
  const [books, setBooks] = useState<Book[]>([]);
  const [searchResults, setSearchResults] = useState<SearchResult[] | null>(
    null,
  );
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [searchLoading, setSearchLoading] = useState(false);

  useEffect(() => {
    loadBooks();
  }, [page]);

  async function loadBooks() {
    setLoading(true);
    try {
      const res = await listBooks({ page, limit: 12, status: "ready" });
      setBooks(res.data || []);
      setTotalPages(res.pagination.total_pages);
    } catch {
      setBooks([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleSearch(query: string) {
    if (!query.trim()) {
      setSearchResults(null);
      return;
    }
    setSearchLoading(true);
    try {
      const res = await searchBooks(query, 10);
      setSearchResults(res.results || []);
    } catch {
      setSearchResults([]);
    } finally {
      setSearchLoading(false);
    }
  }

  return (
    <div>
      <div className="text-center mb-12">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          AI-Powered Exam Preparation
        </h1>
        <p className="text-lg text-gray-600 max-w-2xl mx-auto">
          Practice with AI-generated questions from your textbooks. Get instant
          scoring and detailed feedback on every answer.
        </p>
      </div>

      <div className="mb-8">
        <SearchBar
          onSearch={handleSearch}
          placeholder="Search for books by title, author, or subject..."
          loading={searchLoading}
        />
      </div>

      {searchResults !== null ? (
        <div>
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold">Search Results</h2>
            <button
              onClick={() => setSearchResults(null)}
              className="text-sm text-blue-600 hover:underline"
            >
              Back to library
            </button>
          </div>
          {searchResults.length === 0 ? (
            <p className="text-gray-500 text-center py-8">No results found</p>
          ) : (
            <div className="grid gap-4">
              {searchResults.map((r, i) => (
                <div
                  key={i}
                  className="bg-white rounded-lg shadow p-4 flex gap-4"
                >
                  {r.coverImageUrl && (
                    <img
                      src={r.coverImageUrl}
                      alt={r.title}
                      className="w-16 h-20 object-cover rounded"
                    />
                  )}
                  <div>
                    <h3 className="font-semibold">{r.title}</h3>
                    <p className="text-sm text-gray-600">{r.author}</p>
                    {r.publisher && (
                      <p className="text-xs text-gray-400">{r.publisher}</p>
                    )}
                    {r.description && (
                      <p className="text-sm text-gray-500 mt-1 line-clamp-2">
                        {r.description}
                      </p>
                    )}
                    <span className="text-xs bg-gray-100 px-2 py-0.5 rounded mt-1 inline-block">
                      {r.source}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div>
          <h2 className="text-xl font-semibold mb-4">Available Books</h2>
          {loading ? (
            <div className="text-center py-12 text-gray-500">Loading...</div>
          ) : books.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <p>No books available yet.</p>
              <p className="text-sm mt-2">
                Log in as admin to add books, or register to get started.
              </p>
            </div>
          ) : (
            <>
              <BookGrid books={books} />
              {totalPages > 1 && (
                <div className="mt-8">
                  <Pagination
                    page={page}
                    totalPages={totalPages}
                    onPageChange={setPage}
                  />
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
