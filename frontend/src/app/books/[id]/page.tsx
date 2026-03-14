"use client";

import { useState, useEffect, use } from "react";
import { getBook, listChapters, listQuestions } from "@/lib/api";
import type { Book, Chapter, Pagination as PaginationType } from "@/lib/types";
import Link from "next/link";

export default function BookDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [book, setBook] = useState<Book | null>(null);
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [questionCount, setQuestionCount] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [bookData, chapterData, questionData] = await Promise.all([
          getBook(id),
          listChapters(id, { limit: 100 }),
          listQuestions({ bookId: id, limit: 1 }),
        ]);
        setBook(bookData);
        setChapters(chapterData.data || []);
        setQuestionCount(questionData.pagination.total);
      } catch {
        // Error loading
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  if (loading) return <div className="text-center py-12">Loading...</div>;
  if (!book) return <div className="text-center py-12">Book not found</div>;

  const statusColors: Record<string, string> = {
    ready: "bg-green-100 text-green-800",
    processing: "bg-yellow-100 text-yellow-800",
    pending: "bg-gray-100 text-gray-800",
    failed: "bg-red-100 text-red-800",
  };

  return (
    <div>
      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <div className="flex gap-6">
          {book.coverImageUrl && (
            <img
              src={book.coverImageUrl}
              alt={book.title}
              className="w-32 h-40 object-cover rounded"
            />
          )}
          <div className="flex-1">
            <div className="flex items-start justify-between">
              <h1 className="text-2xl font-bold">{book.title}</h1>
              <span
                className={`px-3 py-1 rounded-full text-sm ${statusColors[book.status] || "bg-gray-100"}`}
              >
                {book.status}
              </span>
            </div>
            <p className="text-gray-600 mt-1">{book.author}</p>
            <div className="flex gap-4 mt-3 text-sm text-gray-500">
              <span>Subject: {book.subject}</span>
              {book.gradeLevels?.length > 0 && (
                <span>Grades: {book.gradeLevels.join(", ")}</span>
              )}
              <span>Chapters: {chapters.length}</span>
              <span>Questions: {questionCount}</span>
            </div>
            {book.status === "ready" && (
              <Link
                href={`/practice/${id}`}
                className="inline-block mt-4 bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700"
              >
                Start Practice
              </Link>
            )}
          </div>
        </div>
      </div>

      <h2 className="text-xl font-semibold mb-4">Chapters</h2>
      <div className="space-y-3">
        {chapters.map((ch) => (
          <div key={ch.id} className="bg-white rounded-lg shadow p-4">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-medium">
                  Chapter {ch.number}: {ch.title}
                </h3>
                {ch.summary && (
                  <p className="text-sm text-gray-500 mt-1">{ch.summary}</p>
                )}
                {ch.topics && ch.topics.length > 0 && (
                  <div className="flex gap-1 mt-2 flex-wrap">
                    {ch.topics.map((t) => (
                      <span
                        key={t}
                        className="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded"
                      >
                        {t}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              <span className="text-xs text-gray-400">
                {ch.wordCount ? `${ch.wordCount} words` : ""}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
