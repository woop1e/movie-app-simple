import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { moviesApi } from '../api/endpoints'
import { EmptyState, ErrorState, Spinner } from '../components/Feedback'
import { Poster, RatingBadge } from '../components/Poster'
import { useApi } from '../hooks/useApi'
import { formatDuration } from '../lib/format'
import type { Movie } from '../types'

export function HomePage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const { data, error, loading, reload } = useApi(
    async () => searchQuery ? { movies: (await moviesApi.search(searchQuery)).movies } : moviesApi.list(),
    [searchQuery],
  )
  const [movies, setMovies] = useState<Movie[]>([])
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [loadMoreError, setLoadMoreError] = useState(false)

  useEffect(() => {
    if (data) {
      setMovies(data.movies)
      setHasMore(data.movies.length === 40)
    }
  }, [data])

  async function loadMore() {
    setLoadingMore(true)
    setLoadMoreError(false)
    try {
      const next = searchQuery ? await moviesApi.search(searchQuery, movies.length) : await moviesApi.list(movies.length)
      setMovies(current => [...current, ...next.movies])
      setHasMore(next.movies.length === 40)
    } catch {
      setLoadMoreError(true)
    } finally {
      setLoadingMore(false)
    }
  }

  if (loading) return <Spinner label="Loading movies…" />
  if (error) return <ErrorState error={error} onRetry={reload} />
  return (
    <div>
      <div className="mb-6">
        <h1 className="page-title">Now showing</h1>
        <p className="mt-1 text-sm text-stone-600">Browse the catalog and discover movies picked for you.</p>
        <form className="mt-4 flex max-w-xl gap-2" onSubmit={event => { event.preventDefault(); setSearchQuery(searchInput.trim()) }}>
          <input className="input" aria-label="Search movies" placeholder="Search titles, genres, or tags" value={searchInput} onChange={event => setSearchInput(event.target.value)} />
          <button className="btn-primary shrink-0" type="submit">Search</button>
          {searchQuery && <button className="btn-secondary shrink-0" type="button" onClick={() => { setSearchInput(''); setSearchQuery('') }}>Clear</button>}
        </form>
      </div>
      {movies.length === 0 ? (
        <EmptyState title="No movies yet">Check back soon — new movies are added regularly.</EmptyState>
      ) : (
        <div className="grid grid-cols-2 gap-6 md:grid-cols-3 lg:grid-cols-4">
          {movies.map((m) => (
            <Link key={m.id} to={`/movies/${m.id}`} className="group card overflow-hidden transition hover:-translate-y-0.5 hover:shadow-md">
              <Poster title={m.title} url={m.posterUrl} className="aspect-[2/3] w-full" />
              <div className="p-4">
                <h2 className="font-semibold text-stone-900 group-hover:text-brand-700">{m.title}</h2>
                <div className="mt-1 flex items-center gap-2 text-xs text-stone-500">
                  <RatingBadge rating={m.rating} />
                  <span>{m.language}</span>
                  <span aria-hidden>·</span>
                  <span>{formatDuration(m.durationMinutes)}</span>
                </div>
                <p className="mt-2 line-clamp-2 text-sm text-stone-600">{m.description}</p>
              </div>
            </Link>
          ))}
        </div>
      )}
      {hasMore && movies.length > 0 && <div className="mt-8 flex flex-col items-center gap-2">
        <button className="btn-secondary" onClick={loadMore} disabled={loadingMore}>
          {loadingMore ? 'Loading…' : 'Load more movies'}
        </button>
        {loadMoreError && <p className="text-sm text-red-700">Could not load more movies. Try again.</p>}
      </div>}
    </div>
  )
}
