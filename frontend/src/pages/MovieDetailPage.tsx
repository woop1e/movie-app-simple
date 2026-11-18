import { Link, useParams } from 'react-router-dom'
import { moviesApi } from '../api/endpoints'
import { activityApi } from '../api/endpoints'
import { BackLink, ErrorState, Spinner } from '../components/Feedback'
import { Poster, RatingBadge } from '../components/Poster'
import { useApi } from '../hooks/useApi'
import { useToast } from '../hooks/useToast'
import { formatDuration, ratingLabel } from '../lib/format'

export function MovieDetailPage() {
  const id = useParams().id ?? ''
  const { data, error, loading, reload } = useApi(() => moviesApi.get(id), [id])
  const toast = useToast()

  if (loading) return <Spinner />
  if (error) return <ErrorState error={error} onRetry={reload} />
  if (!data) return null
  const { movie } = data

  async function record(type: 'view' | 'rating' | 'watchlist', value?: number) {
    try { await activityApi.record(id, type, value); toast.success(type === 'rating' ? 'Rating saved.' : type === 'watchlist' ? 'Added to your watchlist.' : 'View recorded.') }
    catch { toast.error('Log in to save your movie activity.') }
  }

  return (
    <div>
      <BackLink to="/">All movies</BackLink>
      <div className="grid gap-8 md:grid-cols-[240px_1fr]">
        <Poster title={movie.title} url={movie.posterUrl} className="aspect-[2/3] w-full rounded-xl shadow-sm" />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{movie.title}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-stone-600">
            <RatingBadge rating={movie.rating} /><span>{ratingLabel[movie.rating]}</span>
            <span aria-hidden>·</span><span>{movie.language}</span>
            <span aria-hidden>·</span><span>{formatDuration(movie.durationMinutes)}</span>
            {movie.year ? <><span aria-hidden>·</span><span>{movie.year}</span></> : null}
            {movie.category ? <><span aria-hidden>·</span><span>{movie.category}</span></> : null}
          </div>
          <p className="mt-4 max-w-2xl leading-relaxed text-stone-700">{movie.description || 'No description available.'}</p>
          {movie.tags?.length ? <div className="mt-5 flex flex-wrap gap-2">{movie.tags.map(tag => <span key={tag} className="rounded-full bg-stone-100 px-3 py-1 text-xs text-stone-600">{tag}</span>)}</div> : null}
          <section className="card mt-8 max-w-2xl p-5">
            <h2 className="font-semibold">About this movie</h2>
            <p className="mt-1 text-sm text-stone-600">Your ratings, watchlist, and views help shape collaborative recommendations based on viewers with similar activity.</p>
            <Link to="/recommendations" className="btn-primary mt-4 inline-flex">Explore recommendations</Link>
            <div className="mt-3 flex flex-wrap gap-2">
              <button className="btn-secondary" onClick={() => record('view')}>Record view</button>
              <button className="btn-secondary" onClick={() => record('watchlist')}>Add to watchlist</button>
              <button className="btn-secondary" onClick={() => record('rating', 5)}>Rate 5/5</button>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}
