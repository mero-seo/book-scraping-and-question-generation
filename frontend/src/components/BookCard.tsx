import Link from "next/link";
import type { Book } from "@/lib/types";

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800",
  processing: "bg-blue-100 text-blue-800",
  ready: "bg-green-100 text-green-800",
  failed: "bg-red-100 text-red-800",
};

interface BookCardProps {
  book: Book;
}

export default function BookCard({ book }: BookCardProps) {
  const chapterCount = book.toc?.length ?? 0;
  const badgeClass = statusColors[book.status] ?? "bg-gray-100 text-gray-800";

  return (
    <Link
      href={`/books/${book.id}`}
      className="group block rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="mb-3 flex items-start justify-between gap-2">
        <h3 className="text-lg font-semibold text-gray-900 group-hover:text-blue-600 line-clamp-2">
          {book.title}
        </h3>
        <span
          className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${badgeClass}`}
        >
          {book.status}
        </span>
      </div>

      <p className="mb-1 text-sm text-gray-600">{book.author}</p>

      {book.subject && (
        <p className="mb-3 text-sm text-gray-500">
          <span className="font-medium text-gray-700">Subject:</span>{" "}
          {book.subject}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2 text-xs text-gray-500">
        {book.gradeLevels && book.gradeLevels.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {book.gradeLevels.map((level) => (
              <span
                key={level}
                className="rounded bg-gray-100 px-2 py-0.5 text-gray-600"
              >
                {level}
              </span>
            ))}
          </div>
        )}

        {chapterCount > 0 && (
          <span className="ml-auto rounded bg-blue-50 px-2 py-0.5 text-blue-700">
            {chapterCount} {chapterCount === 1 ? "chapter" : "chapters"}
          </span>
        )}
      </div>
    </Link>
  );
}
