import { Link } from 'react-router-dom'
import { activityApi } from '../api/endpoints'
import { EmptyState, ErrorState, Spinner } from '../components/Feedback'
import { Poster, RatingBadge } from '../components/Poster'
import { useApi } from '../hooks/useApi'
import { formatDuration } from '../lib/format'

export function RecommendationsPage() {
  const { data, error, loading, reload } = useApi(() => activityApi.recommendations(), [])
  if (loading) return <Spinner label="Finding movies for you…" />
  if (error) return <ErrorState error={error} onRetry={reload} />
  const movies = data?.movies ?? []
  const isColdStart = data?.algorithm?.startsWith('Cold-start')
  return <div>
    <h1 className="page-title">Recommended movies</h1>
    <p className="mb-6 mt-1 text-sm text-stone-600">
      {isColdStart
        ? 'We’ll personalize these picks as you rate, save, and watch movies.'
        : `Picked from viewers with similar movie activity${data?.similarUsers ? ` · ${data.similarUsers} similar viewers` : ''}.`}
    </p>
    {movies.length === 0 ? <EmptyState title="No recommendations yet">Add favorite genres to your profile or browse more movies.</EmptyState> :
      <div className="grid grid-cols-2 gap-5 sm:grid-cols-3 lg:grid-cols-4">{movies.map(movie => <Link key={movie.id} to={`/movies/${movie.id}`} className="card overflow-hidden transition hover:-translate-y-0.5 hover:shadow-md">
        <Poster title={movie.title} url={movie.posterUrl} className="aspect-[2/3] w-full" />
        <div className="p-4"><h2 className="font-semibold">{movie.title}</h2><div className="mt-2 flex items-center gap-2 text-xs text-stone-500"><RatingBadge rating={movie.rating}/><span>{movie.category}</span><span>·</span><span>{movie.year}</span></div><p className="mt-2 text-xs text-stone-500">{formatDuration(movie.durationMinutes)}</p></div>
      </Link>)}</div>}
  </div>
}
