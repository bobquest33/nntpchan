from django.conf import settings
from django.core.urlresolvers import reverse
from django.http import HttpResponse, Http404
from django.shortcuts import render, get_object_or_404
from django.views import generic


from .models import Post, Newsgroup

from captcha.image import ImageCaptcha
from . import util

captcha = ImageCaptcha(fonts=settings.CAPTCHA_FONTS)


class Postable:
    """
    postable view
    checks captcha etc
    """

    def context_for_get(self, request, defaults):
        defaults['captcha'] = reverse('captcha')
        defaults['refresh_url'] = request.path
        return defaults

    def handle_post(self, request, **kwargs):
        """
        handle post request, implement in subclass
        """
        return None, 'handle_post() not implemented'
    
    def post(self, request, **kwargs):
        ctx = {
            'error' : 'invalid captcha'
        }
        solution = request.session['captcha']
        if solution is not None:
            if 'captcha' in request.POST:
                if request.POST['captcha'].lower() == solution.lower():
                    ctx['msgid'], ctx['error'] = self.handle_post(request, **kwargs)
        request.session['captcha'] = ''
        request.session.save()
        code = 201
        if ctx['error']:
            code = 200
        return HttpResponse(content=render(request, 'frontend/postresult.html', ctx), status=code)

        
class BoardView(generic.View, Postable):
    template_name = 'frontend/board.html'
    context_object_name = 'threads'
    model = Post

    def handle_post(self, request, name):
        """
        make a new thread
        """
        name = 'overchan.{}'.format(name)
        if not util.newsgroup_valid(name):
            return None, "invalid newsgroup: {}".format(name)
        return util.createPost(name, None, request.POST, request.FILES)

    
    def get(self, request, name):
        page = 0
        if 'p' in request.GET:
            page = request.GET['p']
        newsgroup = 'overchan.{}'.format(name)
        try:
            page = int(page or "0")
        except:
            page = 0
        if page < 0:
            page = 0
        try:
            group = Newsgroup.objects.get(name=newsgroup)
        except Newsgroup.DoesNotExist:
            raise Http404("no such board")
        else:
            begin = page * group.posts_per_page
            end = begin + group.posts_per_page - 1
            roots = self.model.objects.filter(newsgroup=group, reference='').order_by('-last_bumped')[begin:end]
            ctx = self.context_for_get(request, {'threads': roots, 'page': page, 'name': newsgroup, 'button': 'new thread'})
            if page < group.max_pages:
                ctx['nextpage'] = reverse('board', args=[name]) + '?p={}'.format(page + 1)
            if page > 0:
                ctx['prevpage'] = reverse('board', args=[name]) + '?p={}'.format(page - 1)
            return render(request, self.template_name, ctx)
        
class ThreadView(generic.View, Postable):
    template_name = 'frontend/thread.html'
    model = Post

    def handle_post(self, request, op):
        """
        make a new thread
        """
        post = get_object_or_404(self.model, posthash=op)
        name = post.newsgroup.name
        if not util.newsgroup_valid(name):
            return None, "invalid newsgroup: {}".format(name)
        return util.createPost(name, post.msgid, request.POST, request.FILES)

    
    def get(self, request, op):
        posts = get_object_or_404(self.model, posthash=op)
        ctx = self.context_for_get(request, {'op': posts, 'button': 'reply'})
        return render(request, self.template_name, ctx)
    
class FrontPageView(generic.View):
    template_name = 'frontend/frontpage.html'
    model = Post

    def get(self, request, truncate=5):
        if truncate <= 0:
            truncate = 5
        posts = self.model.objects.order_by('-posted')[:truncate]
        ctx = {'posts' : posts}
        return render(request, self.template_name, ctx)
    
    
def modlog(request, page):
    if page is None:
        page = 0
    return HttpResponse('mod log page {}'.format(page))

def create_captcha(request):
    solution = util.randstr(7).lower()
    request.session['captcha'] = solution
    request.session.save()
    c = captcha.generate(solution)
    r =HttpResponse(c)
    r['Content-Type'] = 'image/png'
    return r