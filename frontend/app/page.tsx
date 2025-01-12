'use client';

import { useState, useEffect } from "react";

interface SearchResult {
  season: string;
  episode: string;
  start: number;
}

export default function Home() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [selectedResult, setSelectedResult] = useState<SearchResult | null>(null);

  useEffect(() => {
    const delayDebounceFn = setTimeout(() => {
      if (query) {
        fetch(`/api/search?q=${query}`)
          .then((response) => response.json())
          .then((data) => setResults(data))
          .catch((error) => console.error("Error fetching search results:", error));
      }
    }, 500);

    return () => clearTimeout(delayDebounceFn);
  }, [query]);

  const handleImageClick = (result: SearchResult) => {
    setSelectedResult(result);
  };

  return (
    <div className="p-4">
      <input
        type="text"
        placeholder="Search..."
        className="w-full p-2 text-lg border rounded"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      {selectedResult && <CaptionPage {...selectedResult} />}
      {!selectedResult && <SearchResultsPage results={results} onImageClick={handleImageClick} />}
    </div>
  );
}

interface ThumbItem {
  timestamp: number;
}
function CaptionPage(selectedResult: SearchResult) {
  const [thumbs, setThumbs] = useState<ThumbItem[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/thumbs/${selectedResult.season}/${selectedResult.episode}/${selectedResult.start-1000}`)
      .then((response) => response.json())
      .then((data) => {
        setThumbs(data);
      })
      .catch((error) => console.error("Error fetching caption:", error))
      .finally(() => setLoading(false));
  }, [selectedResult]);

  return (
    <div className="mt-4">
      <h2 className="text-2xl font-bold">{`Season ${selectedResult.season}, Episode ${selectedResult.episode}`}</h2>
      <h3 className="text-xl font-bold">{`Start time: ${selectedResult.start} ms`}</h3>
      {loading && <p>Loading...</p>}
      {!loading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-6 lg:grid-cols-6 gap-4 mt-4">
          {thumbs.map((thumb, index) => (
            <img
              key={index}
              src={`/api/thumb/${selectedResult.season}/${selectedResult.episode}/${thumb.timestamp}`}
              alt="Thumbnail"
              className="w-full h-auto rounded"
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface SearchResultsProps {
  results: SearchResult[];
  onImageClick: (result: SearchResult) => void;
}

function SearchResultsPage({ results, onImageClick }: SearchResultsProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-6 lg:grid-cols-6 gap-4 mt-4">
      {results.map((result, index) => (
        <img
          key={index}
          src={`/api/thumb/${result.season}/${result.episode}/${result.start}`}
          alt="Thumbnail"
          className="w-full h-auto rounded cursor-pointer"
          onClick={() => onImageClick(result)}
        />
      ))}
    </div>
  );
}