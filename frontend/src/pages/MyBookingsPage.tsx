import { Link } from 'react-router-dom'
import { activityApi } from '../api/endpoints'
import { EmptyState, ErrorState, Spinner } from '../components/Feedback'
import { useApi } from '../hooks/useApi'

export function MyBookingsPage() {
  const { data, error, loading, reload } = useApi(() => activityApi.history(), [])
  if (loading) return <Spinner label="Loading your activity…" />
  if (error) return <ErrorState error={error} onRetry={reload} />
  const items = data?.history ?? []
  return <div>
    <h1 className="page-title mb-2">My activity</h1>
    <p className="mb-6 text-sm text-stone-600">Your movie views, ratings, watchlist additions, and purchases.</p>
    {items.length === 0 ? <EmptyState title="No activity yet" action={<Link to="/" className="btn-primary">Browse movies</Link>}>Open a movie and start building your recommendation history.</EmptyState> :
      <div className="card divide-y divide-stone-100">{items.map(item => <div key={item.id} className="flex items-center justify-between gap-4 p-4">
        <div><Link to={`/movies/${item.movieId}`} className="font-medium hover:underline">{item.movieTitle || 'Movie'}</Link><p className="mt-1 text-xs capitalize text-stone-500">{item.type}{item.type === 'rating' ? ` · ${item.value}/5` : ''}</p></div>
        <time className="shrink-0 text-xs text-stone-500">{new Date(item.createdAt).toLocaleString()}</time>
      </div>)}</div>}
  </div>
}
