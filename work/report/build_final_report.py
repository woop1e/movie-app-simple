from docx import Document
from docx.shared import Inches, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.table import WD_TABLE_ALIGNMENT, WD_CELL_VERTICAL_ALIGNMENT
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from PIL import Image, ImageDraw, ImageFont
from pathlib import Path

ROOT = Path('/home/v1nt/ALL/Projects/movie-app-assignment')
OUT = Path('/home/v1nt/Documents/Codex/2026-11-11/cv/outputs')
ASSETS = ROOT / 'work/report'
OUT.mkdir(parents=True, exist_ok=True)

NAVY = '#17324D'; BLUE = '#2F6B9A'; TEAL = '#1D8A85'; PALE = '#EAF1F6'; GRAY = '#5F6B75'; LINE = '#D9D9D9'

def font(size, bold=False):
    paths = ['/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf' if bold else '/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf']
    for p in paths:
        if Path(p).exists(): return ImageFont.truetype(p, size)
    return ImageFont.load_default()

def make_chart():
    w,h=1500,700; im=Image.new('RGB',(w,h),'white'); d=ImageDraw.Draw(im)
    d.text((70,32),'Recommendation endpoint latency',font=font(36,True),fill=NAVY)
    d.text((70,82),'Median response time across 8 timed requests after one warm-up',font=font(21),fill=GRAY)
    labels=['10,220 events','50,114 events','100,839 events']; vals=[34.20,136.16,262.51]; maxv=300
    x0,y0,x1,y1=115,170,1380,550
    for tick in range(0,301,50):
        y=y1-(y1-y0)*tick/maxv; d.line((x0,y,x1,y),fill='#E2E8EC',width=2); d.text((45,y-11),str(tick),font=font(18),fill=GRAY)
    bw=210; xs=[280,665,1050]
    for i,(x,v) in enumerate(zip(xs,vals)):
        bh=(y1-y0)*v/maxv; d.rounded_rectangle((x,y1-bh,x+bw,y1),radius=12,fill=BLUE if i<2 else TEAL)
        d.text((x+18,y1-bh-38),f'{v:.1f} ms',font=font(23,True),fill=NAVY)
        d.text((x+20,y1+18),labels[i],font=font(20,True),fill=NAVY)
    d.text((70,635),'Local single-user measurements; see methodology and limitations in the report.',font=font(17),fill=GRAY)
    im.save(ASSETS/'final_latency.png')

def make_model():
    w,h=1500,780; im=Image.new('RGB',(w,h),'white'); d=ImageDraw.Draw(im)
    d.text((70,28),'MongoDB document model',font=font(36,True),fill=NAVY)
    cards={
      'users':(80,150,490,355,[('_id','ObjectId'),('email','unique'),('favoriteGenres','bounded'),('passwordHash','bcrypt')]),
      'events':(545,430,955,660,[('_id','ObjectId'),('userId','reference'),('movieId','reference'),('type / value / time','event facts')]),
      'movies':(1010,150,1420,355,[('_id','ObjectId'),('title / year / category','catalog'),('tags','genres'),('posterUrl','optional')]),
      'sessions':(545,150,955,355,[('_id','token hash'),('userId','reference'),('expiresAt','TTL expiry')])}
    for name,(x,y,xx,yy,rows) in cards.items():
        d.rounded_rectangle((x,y,xx,yy),radius=18,fill='#F8FAFC',outline=BLUE,width=3)
        d.rounded_rectangle((x,y,xx,y+54),radius=16,fill=NAVY)
        d.rectangle((x,y+36,xx,y+54),fill=NAVY)
        d.text((x+20,y+12),name,font=font(26,True),fill='white')
        for n,(key,value) in enumerate(rows):
            yy0=y+76+n*35
            d.text((x+18,yy0),key,font=font(16,True),fill=NAVY)
            d.text((x+220,yy0),value,font=font(16),fill=GRAY)
    # references (events are deliberately separate/unbounded)
    d.line((285,355,620,430),fill=TEAL,width=5); d.polygon([(620,430),(601,418),(606,438)],fill=TEAL)
    d.line((1215,355,880,430),fill=TEAL,width=5); d.polygon([(880,430),(898,418),(894,438)],fill=TEAL)
    d.line((545,250,490,250),fill='#9AA8B2',width=3); d.polygon([(490,250),(501,244),(501,256)],fill='#9AA8B2')
    d.text((80,700),'Users and movies stay independent; events reference both and can grow without enlarging profile documents.',font=font(17),fill=GRAY)
    im.save(ASSETS/'final_data_model.png')

make_chart(); make_model()

doc=Document(); sec=doc.sections[0]; sec.top_margin=Inches(.7); sec.bottom_margin=Inches(.65); sec.left_margin=Inches(.78); sec.right_margin=Inches(.78)
styles=doc.styles
styles['Normal'].font.name='Aptos'; styles['Normal']._element.rPr.rFonts.set(qn('w:eastAsia'),'Aptos'); styles['Normal'].font.size=Pt(10.5); styles['Normal'].font.color.rgb=RGBColor(35,45,54)
styles['Normal'].paragraph_format.space_after=Pt(6); styles['Normal'].paragraph_format.line_spacing=1.12
for sname,size in [('Title',28),('Heading 1',19),('Heading 2',13)]:
    s=styles[sname]; s.font.name='Aptos Display'; s._element.rPr.rFonts.set(qn('w:eastAsia'),'Aptos Display'); s.font.size=Pt(size); s.font.bold=True; s.font.color.rgb=RGBColor(0,0,0); s.paragraph_format.space_before=Pt(12); s.paragraph_format.space_after=Pt(6); s.paragraph_format.keep_with_next=True
title_style_ppr=styles['Title']._element.pPr
if title_style_ppr is not None:
    style_border=title_style_ppr.find(qn('w:pBdr'))
    if style_border is not None: title_style_ppr.remove(style_border)
for sty in styles:
    ppr=sty._element.pPr
    if ppr is not None:
        border=ppr.find(qn('w:pBdr'))
        if border is not None: ppr.remove(border)
styles['Caption'].font.name='Aptos'; styles['Caption'].font.size=Pt(9); styles['Caption'].font.italic=True; styles['Caption'].font.color.rgb=RGBColor(95,107,117)

def shade(cell,color):
    tcPr=cell._tc.get_or_add_tcPr(); shd=OxmlElement('w:shd'); shd.set(qn('w:fill'),color.replace('#','')); tcPr.append(shd)
def borders(table):
    tblPr=table._tbl.tblPr; b=OxmlElement('w:tblBorders')
    for edge in ('top','left','bottom','right','insideH','insideV'):
        e=OxmlElement('w:'+edge); e.set(qn('w:val'),'single'); e.set(qn('w:sz'),'5'); e.set(qn('w:color'),'D9D9D9'); b.append(e)
    tblPr.append(b)
def table(headers, rows, widths=None, font_size=9.2):
    t=doc.add_table(rows=1, cols=len(headers)); t.alignment=WD_TABLE_ALIGNMENT.CENTER; t.autofit=False
    if widths:
        for c,w in zip(t.columns,widths): c.width=Inches(w)
    for i,h in enumerate(headers):
        c=t.rows[0].cells[i]; c.text=h; shade(c,NAVY); c.vertical_alignment=WD_CELL_VERTICAL_ALIGNMENT.CENTER
        for r in c.paragraphs[0].runs: r.font.bold=True; r.font.color.rgb=RGBColor(255,255,255); r.font.size=Pt(font_size)
    for ridx,row in enumerate(rows):
        cells=t.add_row().cells
        if widths:
            for c,w in zip(cells,widths): c.width=Inches(w)
        for i,v in enumerate(row):
            cells[i].text=str(v); cells[i].vertical_alignment=WD_CELL_VERTICAL_ALIGNMENT.CENTER
            if ridx%2: shade(cells[i],'F1F5F8')
            for p in cells[i].paragraphs:
                p.paragraph_format.space_after=Pt(2); p.paragraph_format.space_before=Pt(2)
                for r in p.runs: r.font.size=Pt(font_size)
    borders(t); doc.add_paragraph().paragraph_format.space_after=Pt(1); return t
def para(text='',boldlead=None):
    p=doc.add_paragraph()
    if boldlead and text.startswith(boldlead):
        p.add_run(boldlead).bold=True; p.add_run(text[len(boldlead):])
    else: p.add_run(text)
    return p
def bullet(text):
    p=doc.add_paragraph(style='List Bullet'); p.paragraph_format.space_after=Pt(3); p.add_run(text); return p
def caption(text):
    p=doc.add_paragraph(text,style='Caption'); p.alignment=WD_ALIGN_PARAGRAPH.CENTER
def picture(path,width=6.7):
    p=doc.add_paragraph(); p.alignment=WD_ALIGN_PARAGRAPH.CENTER; p.add_run().add_picture(str(path),width=Inches(width)); return p
def page():
    # Let Word paginate naturally; manual page breaks created sparse pages when
    # a preceding table or figure moved the section heading by a few lines.
    return None

# Header/footer
header=sec.header.paragraphs[0]; header.text='YOZIGO  /  MONGODB FINAL PROJECT'; header.alignment=WD_ALIGN_PARAGRAPH.RIGHT
for r in header.runs: r.font.size=Pt(8); r.font.bold=True; r.font.color.rgb=RGBColor(95,107,117)
foot=sec.footer.paragraphs[0]; foot.alignment=WD_ALIGN_PARAGRAPH.CENTER
foot.add_run('Yozigo · Final Project Report  |  ').font.size=Pt(8)
fld=OxmlElement('w:fldSimple'); fld.set(qn('w:instr'),'PAGE'); foot._p.append(fld)

# Cover / executive summary
p=doc.add_paragraph(); p.paragraph_format.space_before=Pt(70); p.alignment=WD_ALIGN_PARAGRAPH.LEFT
r=p.add_run('FINAL PROJECT REPORT'); r.bold=True; r.font.size=Pt(11); r.font.color.rgb=RGBColor(47,107,154)
p=doc.add_paragraph(style='Title'); p.paragraph_format.space_before=Pt(12); p.add_run('Yozigo Movie Recommendation Platform')
title_ppr=p._p.get_or_add_pPr(); title_border=title_ppr.find(qn('w:pBdr'))
if title_border is not None: title_ppr.remove(title_border)
p=doc.add_paragraph(); p.paragraph_format.space_before=Pt(2); r=p.add_run('MongoDB data model, collaborative filtering, API and performance study'); r.font.size=Pt(15); r.font.color.rgb=RGBColor(95,107,117)
doc.add_paragraph('\n'); para('Prepared for the Final Project assignment'); para('Technology: Go · MongoDB · React · TypeScript'); para('Report date: 18 November 2026')
doc.add_heading('Executive summary',level=1)
para('Yozigo is a working movie discovery application built around MongoDB. It combines account registration and profiles, a catalog expanded with the MovieLens latest-small dataset, interaction history, search, and a user-based collaborative-filtering recommender. The final implementation replaces the earlier newest-title baseline with a similarity-based neighborhood and exposes it through both a JSON API and a React interface.')
para('The local database currently contains 9,748 movie documents, 615 users, and 100,848 interaction events. A repeat import was checked for idempotency. Endpoint measurements on database copies with roughly 10k, 50k, and 100k events show median recommendation response times of 34.20 ms, 136.16 ms, and 262.51 ms respectively. A covering event index reduced the MongoDB query stage from 123 ms to 68 ms in explain output and reduced measured endpoint median by about 9%. These are local, single-request measurements, not a production capacity claim.')
para('The project meets the core requirements for a NoSQL recommendation application: MongoDB collections and indexes, authentication and profiles, catalog CRUD, user events, search, API, sample data, a recommendation algorithm, and a usable interface. The principal dataset limitation is that MovieLens supplies ratings and genres rather than poster images, runtimes, descriptions, or content ratings; the UI displays a generated fallback for those missing fields.')

doc.add_page_break(); doc.add_heading('1. Requirements and delivered features',level=1)
para('The final project brief calls for a recommendation system, database growth and performance measurements, optimization, a working application, setup instructions, and a report with evidence. The implementation maps those requirements to the following features.')
table(['Requirement','Implementation evidence'],[
('NoSQL database','MongoDB 8 stores users, movies, interaction events, and sessions.'),
('Project proposal and data model','Architecture and collection choices are described in Sections 2–3.'),
('Authentication and profile','Registration/login, hashed passwords, opaque sessions, profile genres.'),
('Item catalog','Movie listing, detail, and admin create/update/delete routes.'),
('User history','View, rating, watchlist, and purchase events are stored separately.'),
('Search','Text search plus category filtering; catalog supports pagination.'),
('Recommendation algorithm','Cosine similarity over weighted user interaction vectors.'),
('API and interface','Go JSON endpoints and React pages for catalog, details, activity, and recommendations.'),
('Test data and performance','MovieLens ratings plus seeded demo data; three dataset sizes and index comparison.')], [1.85,4.85])
doc.add_heading('System architecture',level=2)
para('The browser sends JSON requests to the Go API. The API validates input, resolves the signed-in user through a server-side session, and performs document operations with the official MongoDB Go driver. The web container serves the React application and proxies API requests. Docker Compose starts the web, API, and MongoDB services.')
table(['Layer','Technology / role'],[
('Web client','React + TypeScript; movie catalog, detail, activity, recommendation views.'),
('API','Go net/http and MongoDB Go driver; JSON routes and server-side authentication.'),
('Database','MongoDB 8; document collections, compound indexes, TTL expiry.'),
('Local run','Docker Compose; browser at http://localhost:3000 and API at :8080.')],[1.6,5.1])

page(); doc.add_heading('2. MongoDB data model',level=1)
para('The model separates entities with independent lifecycles and puts a small, bounded set of preferences directly on the user. Interactions remain individual documents, so user records do not grow indefinitely as viewing and rating history accumulates.')
picture(ASSETS/'final_data_model.png',6.8); caption('Figure 1. Main collections and references used by the recommendation application.')
table(['Collection','Representative fields','Design reason'],[
('users','_id, name, email, passwordHash, favoriteGenres','Profile and credentials; unique email; genres are a small bounded array.'),
('movies','_id, title, year, category, tags, language, rating, posterUrl','Catalog entity queried independently and paginated.'),
('events','_id, userId, movieId, type, value, createdAt','Appendable interaction stream; references users and movies.'),
('sessions','_id (token hash), userId, expiresAt','Revocable server-side session with TTL expiry.')],[1.0,2.7,3.0],8.8)
doc.add_heading('Embedding and referencing choices',level=2)
para('Favorite genres are embedded in the user profile because the list is small and read with the profile. Movie records are not copied into users or events. Events store ObjectId references so an event is compact and the catalog remains authoritative. This keeps event history unbounded without creating oversized documents. The trade-off is that history screens may need an additional movie lookup to display titles.')
doc.add_page_break()
doc.add_heading('Indexes',level=2)
table(['Collection / index','Purpose'],[
('users.email unique','Fast login lookup and duplicate prevention.'),
('movies title text; title; category + title','Text search and category/title browsing.'),
('movies rating + year; year + title','Catalog filtering and stable newest-first ordering.'),
('events userId + createdAt','Per-user chronological history.'),
('events movieId + type; userId + movieId + type','Interaction lookup and per-movie analytics.'),
('events createdAt + userId + movieId + type + value','Covering recommendation scan and event-time ordering.'),
('sessions.expiresAt TTL','Automatic cleanup of expired session documents.')],[2.5,4.2],8.6)

page(); doc.add_heading('3. Recommendation method',level=1)
para('The recommender uses user-based collaborative filtering. It compares the signed-in user’s positive interaction vector with other users, selects up to 20 neighbors, then scores unseen movies from those neighbors’ positive signals. The system does not rely only on profile genres or newest catalog entries when sufficient collaborative signals exist.')
doc.add_heading('Interaction weighting',level=2)
table(['Interaction','Preference value','Interpretation'],[
('Rating r','(r − 3) / 2','Maps a 1–5 rating to −1…+1; positive ratings contribute to similarity.'),
('Watchlist','0.8','Strong positive implicit signal.'),
('Purchase','1.0','Strongest positive implicit signal.'),
('View','0.2','Weak positive signal.')],[1.35,1.45,3.9])
para('For repeated actions on one user/movie pair, the latest explicit rating takes precedence. Without a rating, the strongest implicit action is retained. Negative and neutral ratings do not contribute to the positive similarity vector, but all interacted-with movies are excluded from that viewer’s recommendations.')
doc.add_heading('Similarity and ranking',level=2)
para('For each other viewer, cosine similarity is computed over movie IDs rated or interacted with positively by both users. A small overlap shrinkage factor, overlap / (overlap + 2), reduces the influence of a match based on only one shared movie. The top 20 resulting neighbors are used. Candidate scores are the neighbor-similarity-weighted mean of their positive preferences. The application removes the target user’s seen items, sorts by score, breaks ties by title, and returns up to 20 movies.')
doc.add_heading('Cold start',level=2)
para('When no positive collaborative candidates exist, the API falls back to catalog items matching the user’s favorite genres, sorted by newest year and title. If the profile has no genre preferences, the same path returns the newest titles. This gives new accounts a usable page while they build interaction history.')
doc.add_heading('Current implementation boundary',level=2)
para('The current recommender scans the event collection and calculates similarities in application memory for each request. The covering index makes the event scan cheaper, but CPU and memory cost still grow with the interaction set. For a larger production system, the next step would be a bounded recent/active interaction window or precomputed user vectors and cached neighbor lists, followed by a quality evaluation on held-out ratings.')

page(); doc.add_heading('4. API, interface and data',level=1)
para('The API exposes the main user workflows through JSON. Authenticated endpoints use an opaque session cookie; stored session IDs are hashes of random tokens, and passwords use bcrypt hashes.')
table(['Endpoint','Purpose'],[
('POST /api/auth/register, /api/auth/login, /api/auth/logout','Account and session lifecycle.'),
('GET /api/auth/me, PUT /api/users/me','Current user and profile update.'),
('GET /api/movies, GET /api/movies/{id}','Paginated catalog and movie detail.'),
('GET /api/search?q=&category=','Text and category search.'),
('POST /api/interactions','Record a view, rating, watchlist action, or purchase.'),
('GET /api/users/me/history','Read the signed-in user’s recent activity.'),
('GET /api/users/me/recommendations','Return collaborative or cold-start recommendations.'),
('POST/PUT/DELETE /api/admin/movies','Catalog management operations.'),
('GET /api/health','Check API and MongoDB connectivity.')],[3.1,3.6],8.8)
doc.add_heading('Expanded test catalog',level=2)
para('GroupLens MovieLens latest-small contributes 9,742 movie metadata records, 610 anonymized user profiles, and 100,836 timestamped rating events. The Yozigo database currently holds 9,748 movies, 615 users, and 100,848 events after combining that dataset with the original demo catalog and the local demonstration account/events. Re-running the importer was checked: the counts remained unchanged.')
para('The importer uses stable source identifiers and event IDs, matches existing titles to preserve richer demo descriptions and poster URLs, and stores synthetic dataset profiles with reserved .invalid email addresses and no passwords. It is safe to use for this local educational demonstration. The MovieLens metadata does not include poster art, runtime, full synopsis, language, or certification; imported films therefore show fallback artwork and unavailable/unknown labels for those fields.')
doc.add_heading('Catalog screenshot',level=2)
picture(ASSETS/'final_catalog.png',6.7); caption('Figure 2. Current Yozigo catalog view with search and MovieLens-backed results. The missing MovieLens images are represented by generated poster tiles.')
doc.add_heading('Movie detail screenshot',level=2)
picture(ASSETS/'final_detail.png',6.45); caption('Figure 3. Movie detail view with tags and controls for recording a view, watchlist action, or rating.')

doc.add_page_break(); doc.add_heading('5. Performance evaluation',level=1)
doc.add_heading('Method',level=2)
para('The endpoint test issued one warm-up request followed by eight timed requests against each local copy database, using the same signed-in demo profile. It recorded full HTTP response time and confirmed that each response returned 20 movies and a collaborative-filtering result. Database sizes were approximately 10k, 50k, and 100k event documents. Each test was run serially on the same development machine; no concurrent load or cold-cache isolation was used.')
para('Environment: Intel Core i5-11400H (6 cores / 12 threads), 15 GiB RAM, MongoDB 8 in Docker, Go API process on the host, both connected through localhost. These numbers describe this setup and workload only.')
table(['Events / copy database','Movies','Median','Mean','Observed p95'],[
('10,220','3,598','34.20 ms','34.78 ms','37.55 ms'),
('50,114','7,487','136.16 ms','136.02 ms','137.88 ms'),
('100,839','9,724','262.51 ms','262.33 ms','263.89 ms')],[1.85,1.05,1.05,1.05,1.7],8.8)
picture(ASSETS/'final_latency.png',6.6); caption('Figure 4. Recommendation endpoint median latency increases with the event count in the measured local copies.')
doc.add_page_break()
doc.add_heading('Index comparison',level=2)
table(['Measurement','Before covering index','With covering index'],[
('MongoDB query explain execution time','123 ms','68 ms'),
('Documents examined','100,839','0'),
('Keys examined','0','100,839'),
('Blocking sort','Yes','No; covered projection'),
('Endpoint median latency','291.45 ms','264.84 ms')],[2.1,2.25,2.25],8.6)
para('The event scan returns all interaction records needed by the current algorithm, so the index does not reduce the number of keys scanned. It instead satisfies event ordering and projection from index keys, avoiding document fetches and a blocking sort. In the explain comparison, MongoDB reports a covered projection; the query-stage time falls by about 44.7%, and the full endpoint median falls by about 9.1%. The remaining endpoint time includes response handling, similarity computation, and movie retrieval.')

page(); doc.add_heading('6. Verification, setup and limitations',level=1)
doc.add_heading('Quick verification performed',level=2)
bullet('Built the Go API and MovieLens importer successfully.')
bullet('Repeated the MovieLens import and verified stable counts: 9,748 movies, 615 users, and 100,848 events.')
bullet('Checked GET /api/health; MongoDB returned status ok.')
bullet('Confirmed the recommendation view returns 20 films and reports 20 similar viewers for the signed-in demo profile.')
bullet('Measured endpoint response time across three event volumes and compared explain plans with and without the covering index.')
para('These are direct build, API, UI, and performance checks. This report does not claim a formal automated integration-test suite, offline ranking-accuracy score, or concurrent capacity result.')
doc.add_heading('Run locally',level=2)
para('From the project root, start the application:')
p=doc.add_paragraph(); r=p.add_run('docker compose up --build'); r.font.name='Consolas'; r.font.size=Pt(9.5)
para('Open http://localhost:3000. The API health route is http://localhost:8080/api/health. The database and app data are local to the Docker setup.')
para('To obtain and import the public sample data from the backend directory:')
p=doc.add_paragraph(); r=p.add_run('mkdir -p ../work/datasets/movielens\ncurl -L https://files.grouplens.org/datasets/movielens/ml-latest-small.zip -o ../work/datasets/movielens/ml-latest-small.zip\nunzip -o ../work/datasets/movielens/ml-latest-small.zip -d ../work/datasets/movielens\ngo run ./cmd/import-movielens'); r.font.name='Consolas'; r.font.size=Pt(8.7)
para('The importer defaults to mongodb://localhost:27017, database yozigo, and the local work/datasets/movielens/ml-latest-small folder. Set MOVIELENS_DIR, MONGODB_URI, or MONGODB_DATABASE to override these values. The download is intentionally kept outside source control.')
doc.add_heading('Limitations and next steps',level=2)
bullet('MovieLens latest-small is a modest development dataset last updated in 2018; it is useful for a classroom prototype, not a current industry catalog or published benchmark.')
bullet('MovieLens supplies sparse explicit ratings and genre tags; generated poster tiles and placeholder metadata can make catalog cards less visually consistent than the seeded demo titles.')
bullet('Recommendation quality is not evaluated with precision@k, recall@k, RMSE, or a held-out split. The report measures runtime only.')
bullet('The current similarity computation is in memory and rescans events on each request. Precomputed vectors or materialized neighborhoods would be needed for larger workloads.')
bullet('The latency run is serial and local. It does not establish multi-user throughput, tail behavior under load, or production deployment performance.')
doc.add_heading('Conclusion',level=2)
para('Yozigo demonstrates the required MongoDB-backed recommendation workflow end to end. The data model keeps the growing interaction stream separate from profiles, the application exposes the required catalog and history operations, and the recommender uses shared behavior to produce personalized candidates. The measured index change improves the database query stage and modestly reduces endpoint latency. The clearest follow-on work is to evaluate ranking quality and replace per-request full-history similarity calculation if the dataset grows beyond this course-project scale.')

page(); doc.add_heading('References',level=1)
refs=[
'GroupLens Research. MovieLens Latest Datasets. https://grouplens.org/datasets/movielens/latest/ (accessed 18 November 2026).',
'Harper, F. Maxwell, and Joseph A. Konstan. “The MovieLens Datasets: History and Context.” ACM Transactions on Interactive Intelligent Systems, 5(4), 2015. https://doi.org/10.1145/2827872.',
'MongoDB, Inc. MongoDB Manual: Indexes. https://www.mongodb.com/docs/manual/indexes/.',
'MongoDB, Inc. MongoDB Manual: TTL Indexes. https://www.mongodb.com/docs/manual/core/index-ttl/.',
'MongoDB, Inc. MongoDB Manual: Text Search. https://www.mongodb.com/docs/manual/text-search/.',
'MongoDB Go Driver. Official Go Driver documentation. https://www.mongodb.com/docs/drivers/go/current/.',
'The Movie Database (TMDB). Image basics. https://developer.themoviedb.org/docs/image-basics.'
]
for i,x in enumerate(refs,1):
    p=doc.add_paragraph(); p.paragraph_format.left_indent=Inches(.25); p.paragraph_format.first_line_indent=Inches(-.25); p.paragraph_format.space_after=Pt(2); p.paragraph_format.line_spacing=1.0
    rr=p.add_run(f'{i}. {x}'); rr.font.size=Pt(9)
para('Dataset note: MovieLens user IDs are anonymized. Review the included README and GroupLens terms before redistribution.')

path=OUT/'Final_Project_MongoDB_Report.docx'; doc.save(path); print(path)
